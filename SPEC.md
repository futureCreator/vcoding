# vCoding — 멀티모델 Issue-to-PR 파이프라인

## 한 줄 요약
입력(이슈/스펙)을 받아, 비싼 모델이 계획을 세우고, 다른 모델이 검증하고, Claude Code가 구현하고, PR까지 자동으로 올리는 파일 기반 개발 파이프라인 CLI.

## 왜 만드는가

### 문제
1. **AI 코딩 도구는 단일 모델 블랙박스다** — Claude Code, Cursor 모두 하나의 모델이 계획부터 구현까지 "알아서" 한다. 자기가 세운 계획을 자기가 검증하면 편향에 빠진다.
2. **컨텍스트가 쌓이면 품질이 떨어진다** — 대화가 길어질수록 토큰 비용은 기하급수적으로 증가하고, 모델은 "Lost in the Middle"에 빠진다.
3. **과정이 보이지 않는다** — 에이전트가 무슨 판단을 했는지, 왜 이렇게 구현했는지 추적할 수 없다.

### 해법
- **비싼 모델이 계획, 다른 모델이 검증, Claude Code가 구현** — 역할별 최적 모델 배치
- **모든 모델 간 통신은 md 파일** — 대화 히스토리 누적 없이, 정제된 산출물만 전달
- **입력에서 PR까지 자동** — 사람은 공정을 YAML로 설계하고, 최종 머지만 판단

## 설계 원칙

### 1. 파일이 곧 프로토콜이다 (File-as-Protocol)
모든 모델 간 통신은 마크다운 파일을 통해 이루어진다. API 히스토리 누적도, 메모리 공유도, 메시지 패싱도 없다.

```
Opus → PLAN.md 작성 → Kimi가 PLAN.md 리뷰 → REVIEW.md → Sonnet이 PLAN.md 확정 → Claude Code가 최종 PLAN.md만 읽고 구현
```

**왜 파일인가:**
- **컨텍스트 최소화**: 각 API 호출은 독립적 단일 턴. 대화 히스토리 O(n²) → 파일 O(n)
- **투명성**: 모든 중간 산출물이 사람이 읽을 수 있는 파일
- **개입 가능성**: 파일을 편집하면 다음 단계에 반영
- **재현성**: 같은 파일로 같은 파이프라인을 다시 돌릴 수 있음

### 2. 이종 모델 교차 검증
같은 모델이 자기 계획을 자기가 검증하면 편향이 동일하다. 서로 다른 모델이 진짜 다른 관점을 제공한다. 계획을 세운 모델(Opus)이 최종 확정까지 하면 리뷰를 무시할 수 있으므로, 확정은 별도 모델(Editor)이 담당한다.

### 3. 구현은 위임한다 (Executor 추상화)
에이전틱 루프를 처음부터 구현하지 않는다. vCoding은 지휘자이지 연주자가 아니다.

### 4. 영어 우선 (English-First)
프롬프트 템플릿과 산출물은 영어로 작성한다. LLM의 코딩 학습 데이터는 영어 중심이므로 영어일 때 품질과 토큰 효율이 높다. 한글 이슈/스펙도 PLAN.md 작성 시 영어로 출력한다.

### 5. 단일 API 게이트웨이 (OpenRouter)
모든 모델을 OpenRouter 하나로 호출한다. API 키 1개, OpenAI-compatible 엔드포인트 1개로 Opus, Sonnet, GPT, Gemini, Kimi 등 전부 사용 가능. provider 구현이 하나면 된다.

## 서브에이전트/팀즈와의 차별점

| | 서브에이전트 / 팀즈 | vCoding |
|---|---|---|
| **통신** | 메모리·메시지 패싱 (휘발) | **md 파일** — 영속, 감사 가능, 편집 가능 |
| **모델** | 같은 모델 N개 복제 | **이종 모델** — 편향 보완 |
| **오케스트레이션** | 프레임워크가 암묵적으로 분배 | **YAML로 개발자가 공정 설계** |
| **재현성** | 세션 종료 시 소멸 | **산출물이 프로젝트에 파일로 남음** |
| **비용** | 에이전트가 알아서 소비 | **단계별 모델·비용 통제** |

## 입력 모드

입력만 다르고, 공정은 동일하다.

```bash
vcoding pick 42           # GitHub 이슈 → TICKET.md 생성
vcoding do SPEC.md        # 스펙 파일을 직접 입력으로 사용
```

모든 입력은 공정의 첫 단계에서 파일로 정규화되어 이후 파이프라인에 전달된다.

## 역할과 공정

### 4가지 역할

| 역할 | 기본 모델 | 하는 일 |
|------|-----------|---------|
| **Planner** | Opus 4.6 | 입력 파일 → 구현 계획 초안 (PLAN.md) |
| **Reviewer** | Kimi K2.5 | PLAN.md 리뷰 (REVIEW.md) |
| **Editor** | Sonnet 4.6 | PLAN.md + REVIEW.md → 최종 PLAN.md 확정 |
| **Auditor** | Codex 5.3 | git diff + PLAN.md → 코드 리뷰 (REVIEW-CODE.md) |

Executor(Claude Code)는 vCoding의 역할이 아니라 외부 도구에 위임하는 것이므로 별도로 둔다.

### 전체 흐름

```
$ vcoding pick 42

🐙 vCoding — #42: Add user authentication
─────────────────────────────────────────
✅ Ticket    TICKET.md                          0.3s
✅ Plan      PLAN.md         opus               $0.45  3.1s
✅ Review    REVIEW.md       kimi               $0.04  2.4s
✅ Revise    PLAN.md         sonnet             $0.12  1.8s
✅ Implement 4 files changed  claude-code        —      48.2s
✅ Test      12 passed        go test            —      1.4s
✅ Audit     REVIEW-CODE.md  codex              $0.06  2.1s
✅ Fix       2 files changed  claude-code        —      12.3s
✅ PR        #87 created                         —      0.5s
─────────────────────────────────────────
✅ Done  $0.73  69s  https://github.com/owner/repo/pull/87
```

### 공정 상세

```
입력 (이슈 / 스펙)
       │
       ▼
┌─────────────┐
│   Ticket    │  입력을 파일로 정규화 (TICKET.md 또는 SPEC.md 그대로 사용)
└──────┬──────┘
       ▼
┌─────────────┐
│  Planner    │  executor: api │ model: Opus
│  PLAN.md    │  입력 파일 + 프로젝트 컨텍스트 → 구현 계획 초안
└──────┬──────┘
       ▼
┌─────────────┐
│  Reviewer   │  executor: api │ model: Kimi K2.5
│  REVIEW.md  │  PLAN.md 리뷰
└──────┬──────┘
       ▼
┌──────────────┐
│   Editor     │  executor: api │ model: Sonnet
│  PLAN.md     │  PLAN.md + REVIEW.md → 최종 계획 확정 (PLAN.md 갱신)
└──────┬───────┘
       ▼
┌─────────────┐
│  Implement  │  executor: claude-code
│             │  최종 PLAN.md만 전달 → 구현 + 커밋
└──────┬──────┘
       ▼
┌─────────────┐
│    Test     │  executor: shell
└──────┬──────┘
       ▼
┌──────────────┐
│   Auditor    │  executor: api │ model: Codex
│REVIEW-CODE.md│  git diff + PLAN.md → 코드 리뷰 (로컬, PR 없이)
└──────┬───────┘
       ▼
┌─────────────┐
│    Fix      │  executor: claude-code
│             │  REVIEW-CODE.md 기반 수정 + 커밋
└──────┬──────┘
       ▼
┌─────────────┐
│  PR 생성    │  vCoding이 gh pr create + 이슈 링크
└──────┬──────┘
       ▼
  인간이 머지 판단
```

### 산출물 파일 맵

| 단계 | 파일 | 설명 |
|------|------|------|
| Ticket | `TICKET.md` | 입력 정규화 (이슈 → 영어) |
| Plan | `PLAN.md` (초안) | 구현 계획 초안 |
| Review | `REVIEW.md` | 리뷰 피드백 |
| Revise | `PLAN.md` (갱신) | 리뷰 반영 최종 계획 |
| Implement | git diff | 코드 변경 |
| Test | `TEST.md` | 테스트 결과 |
| Audit | `REVIEW-CODE.md` | 코드 리뷰 (PR 생성 전, git diff 기반) |
| Fix | git diff | Audit 반영 수정 |

모든 산출물은 `.vcoding/runs/` 하위에 **실행(run) 단위 디렉토리**로 격리 보관한다.

### Run 디렉토리

```
.vcoding/
├── runs/
│   ├── 20260219-1957-fix-auth-bug/     # 타임스탬프 + slug
│   │   ├── TICKET.md
│   │   ├── PLAN.md
│   │   ├── REVIEW.md
│   │   ├── REVIEW-CODE.md
│   │   ├── TEST.md
│   │   └── meta.json                   # 타임스탬프, 비용, 상태, 입력 모드
│   ├── 20260219-2030-add-logging/
│   │   └── ...
│   └── latest -> 20260219-2030-add-logging/   # 현재 진행 중인 run
├── config.yaml
└── pipelines/
```

**네이밍 규칙**: `YYYYMMDD-HHmm-<slug>`
- `pick` 모드: slug = 이슈 번호 + 제목 (`fix-auth-bug`)
- `do` 모드: slug = 스펙 파일명 또는 첫 줄 제목에서 추출 (`add-logging`)

**스펙/이슈는 immutable 입력이다.** 파이프라인 실행 시점의 입력이 run 디렉토리에 스냅샷으로 저장된다. 스펙을 수정했으면 새 run을 돌린다. 이전 run의 산출물은 그대로 남아 참고할 수 있다.

### 컨텍스트 전달

각 스텝은 독립적인 API 단일 턴. 대화 히스토리를 누적하지 않는다.

```
Ticket:    pick → TICKET.md, do → SPEC.md 그대로
Plan:      시스템 프롬프트 + 입력 파일               → PLAN.md (초안)
Review:    시스템 프롬프트 + PLAN.md                  → REVIEW.md
Revise:    시스템 프롬프트 + PLAN.md + REVIEW.md      → PLAN.md (갱신)
Implement: claude -p "$(cat PLAN.md)"                 → 코드
Test:      shell 실행                                 → TEST.md
Audit:     시스템 프롬프트 + PLAN.md + git diff       → REVIEW-CODE.md
Fix:       claude -p "$(cat REVIEW-CODE.md)"          → 코드 수정
```

Audit 1회 → Fix 1회 → PR. 루프 없이 고정 흐름으로 비용·시간 예측 가능.

## 파이프라인 정의

```yaml
# ~/.vcoding/pipelines/default.yaml
name: default

steps:
  - name: Plan
    executor: api
    model: anthropic/claude-opus-4-6
    prompt_template: plan
    input: [TICKET.md]
    output: PLAN.md

  - name: Review
    executor: api
    model: moonshotai/kimi-k2.5
    prompt_template: review
    input: [PLAN.md]
    output: REVIEW.md

  - name: Revise
    executor: api
    model: anthropic/claude-sonnet-4.6
    prompt_template: revise
    input: [PLAN.md, REVIEW.md]
    output: PLAN.md

  - name: Implement
    executor: claude-code
    input: [PLAN.md]

  - name: Test
    executor: shell
    command: "go test ./..."
    output: TEST.md

  - name: Audit
    executor: api
    model: openai/codex-5.3
    prompt_template: code-review
    input: [PLAN.md, git-diff]
    output: REVIEW-CODE.md

  - name: Fix
    executor: claude-code
    input: [REVIEW-CODE.md]

  - name: PR
    type: github-pr
    title_from: TICKET.md
    body_template: pr-summary
```

## Executor

| executor | 설명 | 용도 |
|----------|------|------|
| `api` | OpenRouter API 호출, 단일 턴 | Plan, Review, Revise, Audit |
| `claude-code` | `claude -p` CLI 위임 | Implement, Fix |
| `shell` | CLI 명령 실행 | Test, Lint, Build |

## 기술 스택

- **언어**: Go
- **CLI**: cobra
- **API**: net/http (OpenRouter, OpenAI-compatible)
- **GitHub**: go-github / gh CLI
- **타겟**: macOS / Linux
- **배포**: goreleaser → `brew install vcoding`

## 설정

```yaml
# ~/.vcoding/config.yaml
default_pipeline: default

# OpenRouter — 모든 모델을 단일 API로
provider:
  endpoint: https://openrouter.ai/api/v1
  api_key_env: OPENROUTER_API_KEY

# 역할별 기본 모델
roles:
  planner: anthropic/claude-opus-4-6
  reviewer: moonshotai/kimi-k2.5
  editor: anthropic/claude-sonnet-4.6
  auditor: openai/codex-5.3

# GitHub
github:
  token_env: GITHUB_TOKEN
  default_repo: owner/repo

# Executor
executors:
  claude-code:
    command: claude
    args: ["-p"]
    timeout: 300s

# 언어
language:
  artifacts: en
  normalize_ticket: true

# 프로젝트 컨텍스트
project_context:
  max_files: 20
  max_file_size: 50KB
  include_patterns: ["*.go", "*.rs", "*.ts", "*.py", "*.md"]
  exclude_patterns: ["vendor/", "node_modules/", ".git/"]
```

## CLI

```bash
# 입력 모드
vcoding pick 42              # GitHub 이슈
vcoding do SPEC.md           # 스펙 파일

# 파이프라인 지정
vcoding pick 42 -p quick     # 리뷰 생략 빠른 실행

# 유틸리티
vcoding stats                 # 비용 리포트
vcoding init                  # 초기 설정
```

## 아키텍처

```
vcoding/
├── cmd/vcoding/main.go
├── internal/
│   ├── pipeline/           # 파이프라인 엔진
│   │   ├── engine.go       # 오케스트레이터
│   │   ├── step.go         # 스텝 실행
│   │   └── context.go      # 파일 기반 컨텍스트
│   ├── executor/           # 실행기
│   │   ├── executor.go     # 인터페이스
│   │   ├── api.go          # OpenRouter API
│   │   ├── claudecode.go   # Claude Code CLI
│   │   └── shell.go        # 쉘 명령
│   ├── source/             # 입력 소스
│   │   ├── source.go       # 인터페이스
│   │   ├── github.go       # GitHub Issues (pick)
│   │   └── spec.go         # 스펙 파일 (do)
│   ├── github/             # GitHub 연동
│   │   ├── issue.go        # 이슈 가져오기
│   │   └── pr.go           # PR 생성, 코멘트
│   ├── project/            # 프로젝트 컨텍스트
│   │   ├── scanner.go
│   │   └── git.go
│   ├── cost/tracker.go
│   └── config/config.go
├── prompts/
│   ├── plan.md
│   ├── review.md
│   ├── revise.md
│   ├── code-review.md
│   ├── fix.md
│   └── pr-summary.md
├── pipelines/
│   ├── default.yaml
│   └── quick.yaml          # Plan → Implement → PR (리뷰·감사 생략)
└── go.mod
```

---

