# PLAN.md 검토 의견

## 개요
전반적으로 잘 구성된 구현 계획입니다. 단계별로 명확하게 나뉘어 있고 의존성 관계도 잘 정의되어 있습니다. 아래는 개선 및 보완이 필요한 사항들입니다.

---

## 주요 개선 사항

### 1. Phase 2 - Run 디렉토리: 동시성 안전성
**문제점**: 동시에 여러 `vcoding` 명령이 실행될 경우 Run ID 충돌 가능성
**제안**:
- 타임스탬프에 밀리초/나노초 추가 (`YYYYMMDD-HHmmss-<slug>`)
- 또는 PID 기반 고유 접미사 추가
- `latest` 심볼릭 링크 업데이트 시 race condition 방지 (atomic rename 사용)

### 2. Phase 3 - Executor: 에러 처리 전략
**문제점**: "No loops" 설계 원칙과 executor 실패 시 처리 전략이 불분명
**제안**:
- executor 실패 시 pipeline 중단 (설계 원칙과 일관되게)
- 실패한 run 디렉토리는 `failed/` 서브디렉토리로 이동 또는 `status: failed` 명확히 표시
- 부분 완료된 상태에서의 재시작(`vcoding resume`)은 MVP 이후 고려

### 3. Phase 3 - API Executor: 비용 추출 방식 명확화
**문제점**: 비용 추출 방법이 두 가지로 언급되었으나 우선순위 불분명
**제안**:
```
비용 추출 우선순위:
1. x-openrouter-cost 헤더 (가장 정확)
2. usage.prompt_tokens + usage.completion_tokens * 모델별 가격 (config에 정의)
3. 둘 다 없으면 0으로 기록 + 경고 로그
```

### 4. Phase 4 - Git 정보: Dirty Working Tree 처리
**누락된 사항**: 현재 작업 디렉토리에 uncommitted 변경사항이 있을 때의 처리
**제안**:
- `vcoding pick/do` 실행 전 `git status --porcelain` 확인
- dirty state일 경우:
  - 옵션 A: 경고 후 중단 (안전)
  - 옵션 B: 자동으로 stash (편의)
  - 옵션 C: `--force` 플래그로 무시 가능

### 5. Phase 6 - Pipeline Engine: 입력 파일 처리
**문제점**: `git-diff` 특수 입력 처리가 모호함
**제안**:
- pipeline YAML에서 `input: ["PLAN.md", "git:diff"]` 같은 명확한 문법 사용
- 또는 `input_context: git_diff` 필드 추가
- 지원할 git 컨텍스트 유형 명확히 정의:
  - `git:diff` (staged)
  - `git:diff:base` (base branch와의 diff)
  - `git:log` (최근 커밋)

### 6. Phase 6 - Pipeline Engine: 토큰 한도 관리
**누락된 사항**: 컨텍스트가 너무 클 경우 토큰 한도 초과 가능성
**제안**:
- `max_context_tokens` config 옵션 추가
- 토큰 추정 (rough approximation: 4 chars ≈ 1 token)
- 한도 초과 시 파일 우선순위 기반 truncation 전략

### 7. Phase 7 - GitHub PR: 브랜치 전략
**누락된 사항**: PR 생성 시 브랜치 관리
**제안**:
- `vcoding pick/do` 실행 시 자동 브랜치 생성: `vcoding/<slug>`
- base branch 설정 (config: `github.base_branch`, 기본값: `main`)
- 이미 존재하는 브랜치 처리 전략

### 8. Phase 8 - CLI: 설정 검증
**누락된 사항**: `vcoding init` 이후 설정 유효성 검증
**제안**:
- `vcoding doctor` 명령 추가 (선택사항):
  - OpenRouter API 키 유효성
  - gh CLI 설치/인증 확인
  - git repository 여부 확인
  - config.yaml 유효성 검증

### 9. 테스트 전략 보강
**문제점**: "단위 테스트" 언급만 있고 구체적인 전략 부재
**제안 - 각 phase별 테스트 항목**:
- **Phase 1**: config 로딩 우선순위, 환경변수 치환
- **Phase 2**: run ID 생성 (중복 방지), slug 생성 (특수문자 처리)
- **Phase 3**: API executor timeout, shell executor exit code 처리
- **Phase 6**: pipeline 순환 의존성 감지 (향후 확장을 위해)

### 10. 로깅 전략
**누락된 사항**: 로깅 시스템 명시 없음
**제안**:
- `internal/log` 패키지 추가 (표준 log 또는 log/slog 사용)
- 로그 레벨: ERROR, WARN, INFO, DEBUG (config에서 설정)
- 로그 출력: stderr (stdout은 출력/결과용으로 보존)
- `.vcoding/vcoding.log` 파일에도 기록 (rotate 고려)

---

## 마이너 개선 사항

### 11. Phase 1: Config 구조체
```go
// Validation 메서드 추가 제안
func (c *Config) Validate() error {
    if c.DefaultPipeline == "" {
        return fmt.Errorf("default_pipeline is required")
    }
    // ...
}
```

### 12. Phase 2: Meta 구조체 확장
```go
type Meta struct {
    // ... existing fields ...
    Error       string         `json:"error,omitempty"`       // 실패 시 에러 메시지
    GitBranch   string         `json:"git_branch"`            // 실행 시점 브랜치
    GitCommit   string         `json:"git_commit"`            // 실행 시점 커밋
}
```

### 13. Phase 5: 프롬프트 버전 관리
- 프롬프트 파일에 버전 주석 추가 (e.g., `<!-- version: 1.0 -->`)
- 향후 프롬프트 업데이트 시 호환성 관리

### 14. Phase 10: 사용자 커스터마이징 우선순위 명확화
```
파이프라인 로딩 우선순위:
1. 프로젝트 .vcoding/pipelines/<name>.yaml
2. 사용자 ~/.vcoding/pipelines/<name>.yaml  
3. 내장 embed pipelines/<name>.yaml
```

---

## 구현 순서 조정 제안

| 원래 순서 | 조정 제안 | 이유 |
|-----------|-----------|------|
| Phase 9 (비용 추적) | Phase 3 이후로 이동 | API executor 구현 시 바로 필요 |
| Phase 4 (프로젝트 컨텍스트) | Phase 1 이후로 이동 | Pipeline engine 전에 독립적 구현 가능 |
| Phase 10 (embed 번들) | Phase 5 이후로 이동 | 프롬프트 작성 후 바로 번들링 |

---

## 리스크 및 완화 방안

| 리스크 | 영향도 | 완화 방안 |
|--------|--------|-----------|
| OpenRouter API 한도/장애 | 높음 | timeout 설정, 재시도 없이 명확한 에러 메시지 |
| 큰 diff로 인한 토큰 초과 | 중간 | max_context_tokens 설정, truncation 전략 |
| gh CLI 미설치/미인증 | 높음 | `vcoding doctor` 명령으로 사전 확인 |
| dirty working tree | 중간 | 실행 전 git status 확인, 옵션 제공 |
| 동시 실행 시 run ID 충돌 | 낮음 | 밀리초/나노초 단위 타임스탬프 사용 |

---

## 결론

PLAN.md는 전체적인 설계가 견고하며 구현 단계가 논리적으로 잘 구성되어 있습니다. 위에서 언급한 동시성 안전성, 에러 처리, dirty working tree 처리, 토큰 한도 관리만 보완하면 더욱 robust한 구현이 가능할 것입니다.

핵심 우선순위:
1. **높음**: Run ID 중복 방지, dirty working tree 처리
2. **중간**: 토큰 한도 관리, 브랜치 전략
3. **낮음**: 로깅, doctor 명령 (MVP 이후)
