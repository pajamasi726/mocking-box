# Round 1 대응 — P0(기반·인증)

CRITICAL 은 반박하지 않는다. 리뷰가 맞다. **인증 경계를 절반만 옮겼다** — "어떤 경로가 토큰을 요구하는가"는
옮겨 놓고 "그 토큰을 어디서 읽는가"는 안 옮겼다. 무토큰 프로브 66건이 왜 그걸 못 봤는지도 그대로 재현했다:
같은 66경로를 **수정 전 상태로 되돌려 유효 토큰을 실어** 다시 찔러 봤더니 66건 전부 401 이었고, 무토큰 열은
수정 전후가 글자 하나 다르지 않았다. 401/401 이 공허하다는 지적은 실측으로 확인된다.

| | 무토큰 | 유효토큰(헤더) |
|---|---|---|
| 수정 전 booster, `/api/v1` AUTHENTICATED 66경로 | 401 × 66 | **401 × 66** ← 게이트가 막는다 |
| 수정 후 booster, 같은 66경로 | 401 × 66 | **404 × 66** ← 게이트 통과, 핸들러는 P1~P4 |
| 구(41010) 표본 `GET /api/v1/app-users/me` | 401 | **200** |

같은 토큰을 **쿠키로만** 보내면 구는 401, 수정 전 booster 는 404(=통과)였다. 두 서버가 정반대로 읽고 있었다.

---

## 지적별 대응

| # | 판정 | 근거 |
|---|---|---|
| 1 CRITICAL 토큰 출처 | **반영** | `LegacyHeaderTokenExtractor`(신규, `@Order(HIGHEST_PRECEDENCE)`) 가 `/api/v1` 에서 `Authorization` 헤더를 읽는다. `ApiPathMatcher` 는 손대지 않았고 `/apis`·`/{module}/apis` 는 헤더, 그 밖은 쿠키 그대로다 |
| 2 permitAll 익명 실행 | **반영**(1번이 닫는다) + 영향 전수조사 | 헤더를 읽으면 `appUserId`/`email`/`organizationRole` 이 심긴다. 구 소스 전수: permitAll 49 중 신원을 **실제로 쓰는 건 4개** — 상세 아래 |
| 3 만료 토큰 처리 상이 | **반영**(기능만) | `RestrictedPathPolicy.rejectsInvalidToken()` 신설, `LegacyPathAuthPolicy` 가 `true`. permitAll 에서도 거부한다. **상태코드는 구 400 vs 신 401 로 남는다** — 아래 "반영하지 않은 것" |
| 4 p1b `DROP TABLE` 트랜잭션 | **반영** | 섹션 D 를 `BEGIN;`(`p1b:3637`)/`COMMIT;`(`:3769`) 으로 감쌌다 — `DROP`(`:3658`) 과 재생성(CREATE+IDENTITY+인덱스 4+COMMENT 12) 이 한 트랜잭션에 들어간다. 승인 근거·범위를 같은 자리에 주석으로 명시. 2회 재적용 exit=0 · ERROR 0 · WARNING 0 |
| 5 변경표가 워킹트리와 다름 | **반영** | `p1-schema.sql`·`p1b-schema.sql` 누락분 추가. 표를 `git status --short` 실측(수정 3 + 신규 13 = 16파일)과 일치시키고 "고아 없음" 단언을 그 위에 다시 세웠다 |
| 6 TSV 자기모순 · AC2 계수 | **반영** | TSV 재생성(130행 × 16열). 측정 컬럼을 무토큰/유효토큰/수정전으로 쪼개 모순을 없앴고, booster 는 **130/130 전수 실측**. AC2 표도 130행 전부 계상 |
| 7 excel 경로 누락 + 유령 행 | **반영** + 원인 제거 + 교차검증 | 추출기를 괄호균형 파서로 다시 썼다. 130경로 전수 재추출 → 유령 `GET /api/v1/team` 소멸, `GET /api/v1/team/presentHistories/{teamId}/excel`(`TeamController.kt:70`) 복원. **전 컨트롤러 스윕: 다중행 애노테이션은 이 1건뿐**(경로 2개 이상 담은 `value` 배열도 0건). 아래 교차검증으로 확정 |
| 8 리다이렉트 분기에 얹혀 있음 | **반영** | 정책이 소유한 경로는 `token.fail.redirect-url` 이 채워져도 302 가 아니라 401 이다. 그 값을 채운 상태로 도는 테스트를 붙였다 |
| 9 `resolveRestricted` 무테스트 | **반영** | 3케이스 추가 — ①소유 시 `bypass.url` 미조회 ②정책 0개면 종전 판정 ③다중 정책 `@Order` 타이브레이크(등록 순서와 `@Order` 를 일부러 어긋나게 둬서 `orderedStream()`→`stream()` 퇴행을 잡는다) |
| 10 프리플라이트 | **반영**(P5 설정) | 결함 아님에 동의. `04-changes-P0.md` 의 "P5 판정 규칙 추가분" 3번에 진입점 게이트웨이 고정을 명시 |
| 11 죽은 규칙 · 과장된 주석 | **반영**(주석 정정) | `:49` 는 "정리하지 말 것" 주석을 달아 남겼다. "순서를 바꾸면 판정이 바뀐다"는 지적대로 이 규칙집합에선 성립하지 않는다 — 그렇게 고쳐 적었다 |

**리뷰어 정정 2건** — 둘 다 반영. ①401 바디 차이의 시한을 P5 → **P6 이전**으로 옮겨 적었고, 만료 토큰의
400 vs 401 도 같은 "에러 봉투" 항목으로 묶었다. ②500→404 는 "조용히 넘어간다"가 아니라 **판정 기준이
통과시킨다**로 고쳐 쓰고, **`구 500 → 신 404` 를 불합격**으로 P5 판정 규칙에 넣었다.

### 2번 상세 — permitAll 에서 신원이 실제로 바디·write-set 을 가르는 경로
| 경로 | 신원이 없으면 |
|---|---|
| `POST /api/v1/organizations/{organizationId}/invitations` | `AUTH_REQUIRED` 로 **400**(`OrganizationController.kt:249`). permitAll 인데 신원 필수 — 이건 리플레이가 잡는다(200→400) |
| `POST /api/v1/store/register` | `appUser?.id` 가 크롤 요청 Kafka 메시지에 실린다 → **write-set 이 달라진다**(`StoreFacade:100`) |
| `GET /api/v1/posts` · `GET /api/v1/stores/{storeId}/posts/statistics` | 값을 넘기기만 하고 `PostRepositoryImpl:50,125` 이 쓰지 않는다 → 현재 코드에선 응답 동일 |

즉 지적의 예시였던 `GET /api/v1/posts/**` 는 실제로는 신원 무관이었지만, 다른 두 경로에서 바디와
write-set 이 갈린다. 어느 쪽이든 1번을 고치면서 같이 닫혔다.

## 반영하지 않은 것 (1건, 반박 아니라 이월)
**만료 토큰의 상태코드 400(구) vs 401(신).** `TokenVerificationFilter` 의 공용 catch 가 상태를 401 로 고정해
두었고, 거기를 건드리면 기존 9모듈 전체의 인증 실패 응답이 같이 움직인다. 지적 3번의 **기능적 차이(구는
거부, 신은 서빙)는 닫았고**, 남은 코드 차이는 401 바디 차이(`AUTH-001` vs `90008`)와 같은 성격이라
**P6 전 결정 항목**으로 함께 올렸다. 실측 근거: 구 `400 {"errors":[{"code":"AUTH-006"...}]}` / 신 `401 {"code":"90006"}`.

## 재검증

**테스트** — `./gradlew test --rerun-tasks`(booster 전체) **689/0/0/0** (초안 672 → +17).
`duplicateClassCheck` 그린. 게이트웨이 **76/0**, `RouteTableTest` 22개 통과, `Routes.java:34` = `ServiceId.LEGACY` 유지.

**추출 교차검증** — 새 추출기 130행을 **독립적으로 만들어진 `P-1-endpoint-table-map.tsv`** 와 대조했다.
그 표는 P-1 단계에서 다른 방법으로 뽑은 것이고 처음부터 excel 경로를 갖고 있었으며(`:56`)
`/api/v1/team` 은 클래스 레벨 접두로 `엔드포인트 아님`(`:139`, 출처 `TeamController.kt:26`)이라고 적어 뒀다.
**실엔드포인트 130 ↔ 130, 양방향 차집합 0.** 같은 태스크 안의 두 산출물이 어긋나 있었다는 것 자체가
신호였는데 초안이 그걸 못 봤다 — 이제 둘이 일치한다.

**변이 검사** — 새 테스트가 실제로 이 결함을 잡는지 확인했다. ①추출기 `supports()`→`false` ②`@Order` 제거
③`rejectsInvalidToken()`→`false` 를 동시에 넣고 돌리니 **6건 실패**(추출기 주입 순서 / 헤더 통과 / 쿠키 거부 /
permitAll 신원 / permitAll 만료 거부 / 소유 일치). 되돌린 뒤 전부 통과.

**개발계 실측** — booster 로컬(8080)에서 130경로를 **수정 전·후 각각** 무토큰·유효토큰 두 벌로 스윕(총 520 호출).
구(41010)는 AUTHENTICATED 67경로 무토큰(전부 401) + 유효토큰 표본 3건 + 만료토큰 3건.
구 유효토큰은 `POST /api/v1/jwt/refresh`(permitAll, DB 전용)로 발급했고 표본 경로는 순수 조회임을 소스로 확인했다.
만료 토큰은 골든셋 캡처분(exp 2026-07-15)을 그대로 썼다. **토큰·시크릿 값은 어디에도 출력하지 않았다.**

| 프로브 | 구 | 신(수정 후) | 신(수정 전) |
|---|---|---|---|
| `GET /api/v1/app-users/me` 헤더 유효토큰 | 200 | 404(게이트 통과) | **401** |
| `GET /api/v1/app-users/me` 쿠키만 | **401** | **401** | 404(쿠키를 읽었다) |
| `GET /api/v1/health` permitAll + 만료토큰 | **400** `AUTH-006` | **401** `90006` | **200**(삼키고 진행) |
| `/api/v2/boost1/**` 헤더 유효토큰 | — | 401(쿠키 출처 유지) | 401 |
| `/user/apis/unrestricted/x` · `/external/apis/swagger` | — | 404 · 404 | 404 · 404 |
| `/post/apis/x` | — | 401 | 401 |

**DDL** — p1b 2회 재적용 exit=0 · ERROR 0 · WARNING 0, `BEGIN`/`COMMIT` 로그 확인. 신규 PG BASE TABLE **144 불변**,
`boost_message_delivery` 컬럼 위치 16·인덱스 7 유지(가드가 걸려 `DROP` 은 실행되지 않았다).

**불변조건** — 컨테이너 **91 running**(착수 시와 동일), `legacy-service`·`legacy-bridge` 둘 다 Up 3 days,
최근 종료된 컨테이너 0건(가장 최근 exit 이 2일 전), 운영 RDS 접근 0, 게이트웨이 라우트 무변경,
`medilawyer-boot` 무수정, 커밋·푸시 0. 구에 남긴 쓰기는 `jwt/refresh` 가 자기 `token` 행을 갱신한 것 하나다.

## 남은 미판정
① 구·신 **JWT 서명키가 같은지 확인하지 못했다.** 구는 config-server+Vault 에서 받고
`/legacy-service/workstation,git,vault` 조회가 `propertySources: []` 로 비어 온다. 이번 실측은 각 서버가
자기 키로 서명한 토큰을 썼으므로 "각 서버가 헤더를 읽는가"는 증명되지만 "구 토큰을 신이 검증하는가"는 아니다 —
그건 원래 **AC5(P3)** 의 4조합 항목이고, P3 착수 시 키 소재 확정이 선행 조건이다.
② permitAll 48경로의 **구 실측은 여전히 없다**(유효 토큰을 실으면 구 컨트롤러가 실제로 실행된다 — 불변조건 5).
booster 쪽 전수 + 핸들러 없는 하위경로 프로브로 대신했고, P1~P4 리플레이에서 자연히 해소된다.
③ P5 판정 규칙 추가분은 `04-changes-P0.md` 에 적었지 `검증방식-리플레이-차등검증.md` 자체는 고치지 않았다.
그 문서는 다른 태스크가 공유하는 절차 원본이라 이번 범위에서 건드리지 않았다 — **P5 착수 시 반영이 착수 조건**이다.

STATUS: REVISED
