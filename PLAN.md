## 구현 계획

### Phase 0: 프로젝트 부트스트랩

**목표**: Go 모듈 초기화, CLI 뼈대, 빌드 확인

1. `go mod init github.com/epmk/vcoding`
2. cobra CLI 뼈대 생성
   - `cmd/vcoding/main.go` — rootCmd + version
   - `vcoding init` — `~/.vcoding/config.yaml` 생성 (대화형 or 기본값)
   - `vcoding pick <issue>` — stub (TODO 출력)
   - `vcoding do <spec>` — stub (TODO 출력)
   - `vcoding stats` — stub (TODO 출력)
3. `go build ./cmd/vcoding` 확인
4. `.gitignore` 작성 (바이너리, `.vcoding/runs/`)

**산출물**: `vcoding version` 실행 가능한 바이너리

---

### Phase 1: 설정 시스템

**목표**: config.yaml 로딩, 파이프라인 YAML 파싱

**파일**: `internal/config/config.go`

1. Config 구조체 정의
   ```go
   type Config struct {
       DefaultPipeline  string           `yaml:"default_pipeline"`
       Provider         ProviderConfig   `yaml:"provider"`
       Roles            RolesConfig      `yaml:"roles"`
       GitHub           GitHubConfig     `yaml:"github"`
       Executors        ExecutorsConfig  `yaml:"executors"`
       Language         LanguageConfig   `yaml:"language"`
       ProjectContext   ProjectCtxConfig `yaml:"project_context"`
       MaxContextTokens int              `yaml:"max_context_tokens"` // default: 80000
   }

   // Validate checks required fields
   func (c *Config) Validate() error {
       if c.DefaultPipeline == "" {
           return fmt.Errorf("default_pipeline is required")
       }
       // ...
   }
   ```
2. 설정 로딩 우선순위: 프로젝트 `.vcoding/config.yaml` → `~/.vcoding/config.yaml` → 기본값
3. 환경변수 해석: `api_key_env: OPENROUTER_API_KEY` → `os.Getenv("OPENROUTER_API_KEY")`
4. `vcoding init` 구현: 기본 config.yaml + default pipeline YAML 생성

**파일**: `internal/pipeline/pipeline.go`

5. Pipeline YAML 파싱
   ```go
   type Pipeline struct {
       Name  string `yaml:"name"`
       Steps []Step `yaml:"steps"`
   }
   type Step struct {
       Name           string   `yaml:"name"`
       Executor       string   `yaml:"executor"`
       Model          string   `yaml:"model,omitempty"`
       PromptTemplate string   `yaml:"prompt_template,omitempty"`
       Input          []string `yaml:"input"`
       Output         string   `yaml:"output,omitempty"`
       Command        string   `yaml:"command,omitempty"`
       Type           string   `yaml:"type,omitempty"`
       TitleFrom      string   `yaml:"title_from,omitempty"`
       BodyTemplate   string   `yaml:"body_template,omitempty"`
   }
   ```
6. `pipelines/default.yaml`, `pipelines/quick.yaml` 내장 (embed)

**테스트**: config 로딩, pipeline 파싱 단위 테스트

---

### Phase 2: Run 디렉토리 & 입력 소스

**목표**: run 디렉토리 생성, 입력을 파일로 정규화

**파일**: `internal/run/run.go`

1. Run 구조체
   ```go
   type Run struct {
       ID      string    // "20260219-195732-123-fix-auth-bug"
       Dir     string    // ".vcoding/runs/20260219-195732-123-fix-auth-bug/"
       Meta    Meta
   }
   type Meta struct {
       StartedAt  time.Time    `json:"started_at"`
       InputMode  string       `json:"input_mode"`       // "pick" | "do"
       InputRef   string       `json:"input_ref"`        // issue number or spec path
       Status     string       `json:"status"`           // "running" | "completed" | "failed"
       Steps      []StepResult `json:"steps"`
       TotalCost  float64      `json:"total_cost"`
       Error      string       `json:"error,omitempty"`  // failure message
       GitBranch  string       `json:"git_branch"`       // branch at execution time
       GitCommit  string       `json:"git_commit"`       // commit at execution time
   }
   ```
2. Run 디렉토리 생성: `YYYYMMDD-HHmmss-<ms>-<slug>` (밀리초 포함으로 동시 실행 시 충돌 방지)
3. `latest` 심볼릭 링크 관리 — atomic rename (`os.Rename`) 사용으로 race condition 방지
4. `meta.json` 쓰기/읽기

**파일**: `internal/source/source.go`, `github.go`, `spec.go`

5. Source 인터페이스
   ```go
   type Source interface {
       Fetch(ctx context.Context) (*Input, error)
   }
   type Input struct {
       Title   string
       Body    string
       Slug    string
       Mode    string // "pick" | "do"
       Ref     string // issue number or file path
   }
   ```
6. `GitHubSource.Fetch()` — `gh issue view <num> --json` 으로 이슈 가져오기 → TICKET.md 작성
7. `SpecSource.Fetch()` — 파일 읽기, 첫 줄에서 slug 추출

**테스트**: run 디렉토리 생성, slug 생성, spec 파싱 단위 테스트

---

### Phase 3: Executor 구현

**목표**: 3가지 executor (api, claude-code, shell) 구현

**파일**: `internal/executor/executor.go`

1. Executor 인터페이스
   ```go
   type Executor interface {
       Execute(ctx context.Context, req *Request) (*Result, error)
   }
   type Request struct {
       Step       pipeline.Step
       RunDir     string
       InputFiles map[string]string // filename → content
       GitDiff    string            // for audit step
   }
   type Result struct {
       Output     string
       Cost       float64
       Duration   time.Duration
       TokensIn   int
       TokensOut  int
   }
   ```

**파일**: `internal/executor/api.go`

2. OpenRouter API executor
   - OpenAI-compatible chat completion API 호출
   - 시스템 프롬프트 = `prompts/<template>.md` 내장 파일
   - 유저 메시지 = input 파일 내용 결합
   - 응답 → output 파일 저장
   - 비용 추출 우선순위:
     1. `x-openrouter-cost` 응답 헤더 (가장 정확)
     2. `usage.prompt_tokens` + `usage.completion_tokens` × 모델별 단가 (config에 정의)
     3. 둘 다 없으면 0 기록 + WARN 로그
   - 스트리밍 없음 (단일 턴이므로 non-streaming으로 충분)

**파일**: `internal/executor/claudecode.go`

3. Claude Code executor
   - `claude -p --output-format json` 실행
   - stdin으로 프롬프트 전달: input 파일 내용
   - stdout/stderr 캡처
   - timeout 적용 (config에서)

**파일**: `internal/executor/shell.go`

4. Shell executor
   - `exec.CommandContext`로 command 실행
   - stdout → TEST.md 저장
   - exit code 캡처, 0이 아니면 에러

**테스트**: api executor mock 테스트, shell executor 단위 테스트

---

### Phase 4: 프로젝트 컨텍스트

**목표**: Planner에게 전달할 프로젝트 구조/코드 수집

**파일**: `internal/project/scanner.go`

1. 프로젝트 파일 수집
   - `include_patterns`, `exclude_patterns` 기반 glob
   - `max_files`, `max_file_size` 제한
   - 파일 목록 + 주요 파일 내용 → 마크다운 포맷

**파일**: `internal/project/git.go`

2. Git 정보 수집
   - 현재 브랜치, 최근 커밋
   - `git diff` (Audit 스텝용)
   - base branch와의 diff
3. Dirty working tree 확인 (`git status --porcelain`)
   - `vcoding pick/do` 실행 전 uncommitted 변경사항 감지
   - dirty state일 경우 경고 후 중단 (기본)
   - `--force` 플래그로 무시 가능

**테스트**: scanner 단위 테스트

---

### Phase 5: 프롬프트 템플릿

**목표**: 각 역할의 시스템 프롬프트 작성 (embed)

**파일**: `prompts/` 디렉토리 (Go embed)

1. `plan.md` — Planner 시스템 프롬프트
   - 입력: TICKET.md (또는 SPEC.md) + 프로젝트 컨텍스트
   - 출력 포맷: 구조화된 PLAN.md (목표, 변경 파일 목록, 구현 단계, 엣지 케이스)
2. `review.md` — Reviewer 시스템 프롬프트
   - 입력: PLAN.md
   - 출력 포맷: REVIEW.md (문제점, 개선사항, 승인/거부)
3. `revise.md` — Editor 시스템 프롬프트
   - 입력: PLAN.md + REVIEW.md
   - 출력 포맷: 갱신된 PLAN.md
4. `code-review.md` — Auditor 시스템 프롬프트
   - 입력: PLAN.md + git diff
   - 출력 포맷: REVIEW-CODE.md (버그, 보안, 스타일, 수정 지시)
5. `pr-summary.md` — PR body 생성 템플릿

---

### Phase 6: 파이프라인 엔진

**목표**: 스텝을 순서대로 실행하는 오케스트레이터

**파일**: `internal/pipeline/engine.go`

1. Engine 구조체
   ```go
   type Engine struct {
       Config    *config.Config
       Pipeline  *Pipeline
       Executors map[string]executor.Executor
       Run       *run.Run
   }
   ```
2. `Engine.Execute(ctx, input)` — 메인 루프
   - 각 step 순회
   - input 파일 로딩 (run 디렉토리에서)
   - 특수 입력 처리 (`git:diff`, `git:diff:base`, `git:log`)
     - pipeline YAML에서 `input: ["PLAN.md", "git:diff"]` 문법 사용
   - executor 선택 및 실행
   - output 파일 저장 (run 디렉토리에)
   - meta.json 갱신 (step 결과, 비용, 소요 시간)
   - 실시간 진행 출력 (스펙의 터미널 UI 포맷)

**파일**: `internal/pipeline/context.go`

3. 스텝 간 파일 컨텍스트 관리
   - input 목록에서 파일 읽기
   - 프로젝트 컨텍스트 결합 (Plan 스텝만)
   - 토큰 추정 (rough: 4 chars ≈ 1 token) 및 `max_context_tokens` 초과 시 파일 우선순위 기반 truncation

**파일**: `internal/pipeline/display.go`

4. 터미널 진행 표시
   - 스텝별 상태 (spinner → 완료/실패)
   - 모델명, 비용, 소요 시간 표시
   - 최종 요약 (총 비용, 총 시간, PR URL)

---

### Phase 7: GitHub 연동 & PR 생성

**목표**: 이슈 가져오기, PR 생성

**파일**: `internal/github/issue.go`

1. `gh issue view` 래핑 — 이슈 제목, 본문, 라벨 가져오기

**파일**: `internal/github/pr.go`

2. PR 생성
   - `gh pr create` 래핑
   - 제목: TICKET.md에서 추출
   - 본문: pr-summary 템플릿으로 생성 (PLAN.md + 변경 파일 요약)
   - 이슈 링크: `Closes #42`
   - 브랜치 전략:
     - `vcoding pick/do` 실행 시 자동 브랜치 생성: `vcoding/<slug>`
     - base branch: config `github.base_branch` (기본값: `main`)
     - 브랜치 이미 존재 시 에러로 중단 (의도치 않은 덮어쓰기 방지)

---

### Phase 8: CLI 커맨드 완성

**목표**: pick, do, stats 커맨드 연결

1. `vcoding pick <issue-number>` [-p pipeline] [--force]
   - dirty working tree 확인 (--force로 무시 가능)
   - GitHubSource → Run 생성 → Engine.Execute
2. `vcoding do <spec-file>` [-p pipeline] [--force]
   - dirty working tree 확인 (--force로 무시 가능)
   - SpecSource → Run 생성 → Engine.Execute
3. `vcoding stats`
   - `.vcoding/runs/*/meta.json` 순회
   - 총 비용, run 수, 평균 비용, 모델별 비용 출력
4. `-p` 플래그로 파이프라인 지정 (기본: default)
5. `vcoding doctor` (선택사항, MVP 이후)
   - OpenRouter API 키 유효성 확인
   - gh CLI 설치/인증 확인
   - git repository 여부 확인
   - config.yaml 유효성 검증 (Config.Validate())

---

### Phase 9: 비용 추적

**목표**: 스텝별 비용 기록

**파일**: `internal/cost/tracker.go`

1. OpenRouter 응답에서 비용 추출 (Phase 3 API executor와 함께 구현)
   - `x-openrouter-cost` 헤더 우선
   - 없으면 `usage.prompt_tokens` + `usage.completion_tokens` × 모델별 단가
   - 둘 다 없으면 0 기록 + WARN 로그
2. meta.json에 스텝별 비용 기록
3. `vcoding stats`에서 집계

---

### Phase 11: 로깅

**목표**: 일관된 로그 출력

**파일**: `internal/log/log.go`

1. log/slog 기반 구조화 로그
2. 로그 레벨: ERROR, WARN, INFO, DEBUG (config `log_level`에서 설정)
3. stderr 출력 (stdout은 사용자 결과/출력 전용으로 보존)
4. `.vcoding/vcoding.log` 파일에도 기록

---

### Phase 10: 내장 파이프라인 & 프롬프트 번들

**목표**: default.yaml, quick.yaml, 프롬프트 파일을 Go embed로 번들

1. `//go:embed pipelines/*.yaml` → 내장 파이프라인
2. `//go:embed prompts/*.md` → 내장 프롬프트
3. 사용자가 `~/.vcoding/pipelines/`에 커스텀 파이프라인 추가 가능
4. 사용자 파일이 있으면 내장보다 우선

---

### 구현 순서 요약

| 순서 | Phase | 핵심 산출물 | 의존성 |
|------|-------|-------------|--------|
| 1 | Phase 0 | CLI 뼈대, `go build` | 없음 |
| 2 | Phase 1 | config 로딩, pipeline 파싱 | Phase 0 |
| 3 | Phase 4 | 프로젝트 컨텍스트, git 정보 | Phase 1 |
| 4 | Phase 2 | run 디렉토리, 입력 소스 | Phase 1 |
| 5 | Phase 3 | 3가지 executor | Phase 1 |
| 6 | Phase 9 | 비용 추적 | Phase 3 |
| 7 | Phase 5 | 프롬프트 템플릿 | 없음 |
| 8 | Phase 10 | embed 번들 | Phase 5 |
| 9 | Phase 6 | 파이프라인 엔진 | Phase 2, 3, 4, 5 |
| 10 | Phase 7 | GitHub 연동 | Phase 2 |
| 11 | Phase 8 | CLI 커맨드 완성 | Phase 6, 7 |
| 12 | Phase 11 | 로깅 | 없음 (초기부터 점진적 적용) |

Phase 3, 4, 5는 서로 독립적이므로 병렬 구현 가능.
Phase 11 (로깅)은 전 단계에 걸쳐 점진적으로 적용.

---

### MVP 기준 (Phase 0~8 완료 시)

- `vcoding do SPEC.md` 로 스펙 → PLAN.md → REVIEW.md → 최종 PLAN.md → 구현 → 테스트 → 코드 리뷰 → 수정 → PR 전체 흐름 동작
- `vcoding pick 42` 로 GitHub 이슈 기반 동일 흐름 동작
- `.vcoding/runs/` 에 모든 산출물 보관
- 터미널에 진행 상황 실시간 표시

