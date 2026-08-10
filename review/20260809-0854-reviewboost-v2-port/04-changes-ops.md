# 변경 요약: 컷오버 선행 — 예외 알림 실연결(4앱) + 404 알림 이식

worktree `legalcare-renew-prodsync-wt` / `cacaec5` 위. 커밋 안 함.

## 지적 대응
| # | 지적 | 대응 |
|---|---|---|
| 1 | dev 웹훅은 env 가 아니라 profile 하드코딩이 관례 | **반영**. gateway `application-dev.yml` 에 dev 웹훅 기본값 추가(앱 3종과 동일 값), `application.yml` 은 빈 기본값 유지(운영=env). `.env.example`·compose 는 **override 전용**으로 문구 정정, `SLACK_NOTFOUND_*` 는 삭제. 앱 3종 dev yml 복원분은 그대로 두고 내가 넣었던 `notfound-notify` 키만 제거. |
| 2 | 404 알림은 이미 구현돼 있었다 / 중복 구현 | **반영**. 내 구현(`NotFoundSlackNotifier`·`NotFoundPathNormalizer` × 4앱 + 테스트, 16파일) **전량 삭제**하고 원본을 이식. 게이트웨이 `RouteMissAlerter` 는 원본과 **byte-identical**, 앱 3종 `isNotifiable404`/`dedupeKey404` 는 원본과 **문자 단위 동일**(스크립트로 대조). |
| 3 | 원본을 찾아 diff 를 보고 맞춰라 | 완료. 다만 **사실관계 정정** — 아래. |

**사실관계 정정(중요):** 원본 커밋들은 유실된 게 아니고 내 구현으로 대체된 것도 아니다. 구 서비스별 레포에만 있다 — `549f9dd`는 `remotes/booster-api/develop`, `569f91a`·`9b98138`·`df042fb`는 `remotes/legalcare-gateway/develop`. 넷 다 `git merge-base --is-ancestor <c> HEAD` = **NO**, `git ls-tree HEAD | grep RouteMissAlerter` = **비어 있음**. renew 모노레포로 **애초에 이관되지 않은 것**이고, 이번에 이관했다. 내 잘못은 "지웠다"가 아니라 **다른 리모트를 안 찾아보고 새로 짠 것**이다.

## 변경 파일
| 파일 | 한 줄 요약 | R# |
|---|---|---|
| `apps/gateway-app/**/alert/RouteMissAlerter.java` (신규) | 구 게이트웨이 레포에서 **원본 그대로** 이식(GW 404) | B |
| `apps/{booster,reviewed,lawkit}/**/advice/ControllerExceptionAdvice.java` | 원본 404 블록 + `isNotifiable404`/`dedupeKey404` 이식, `@PostConstruct` 상태 로그 | B, A4 |
| `apps/gateway-app/**/alert/GatewayErrorSlackNotifier.java` (신규) | 게이트웨이 예외 알림 **신설**(원본에 없던 부분), 상태 로그 | A3, A4 |
| `apps/gateway-app/**/alert/ErrorNotifyFilter.java` (신규) | 필터 체인 예외 포착 → 알림 후 그대로 재throw | A3 |
| `apps/gateway-app/**/proxy/ProxyFilter.java` | 라우트 미매칭→`RouteMissAlerter`(원본 호출 형태), 업스트림 실패→예외 알림 | A3, B |
| `apps/gateway-app/**/config/GatewayConfig.java` | 알림기 주입 + ErrorNotifyFilter 등록 | A3 |
| `apps/gateway-app/src/main/resources/application-dev.yml` | **dev 웹훅 하드코딩**(관례) + `gateway.alert.*` 명시 | A1 |
| `apps/gateway-app/src/main/resources/application.yml` | `slack.error-notify` 빈 기본값(운영=env) | A1 |
| `apps/{3}/app/.../application-dev.yml` | 내가 넣었던 `notfound-notify` 키 제거(복원분은 유지) | A1 |
| `docker-compose.yml`, `deploy/dev/docker-compose.yml`, `.env.example` ×2 | override 전용으로 정정, `SLACK_NOTFOUND_*` 삭제 | A1 |
| `docs/deploy/dev-cutover.md` | "dev 는 env 불필요" + 두 404 신호 구분표 | A1 |
| 테스트 4개(신규 3·수정 1) | 포맷 잠금·노이즈 제외·dedupe·상한·재throw·배선 | A3, B |
| ~~`NotFoundSlackNotifier`·`NotFoundPathNormalizer` ×4앱 + 테스트~~ | **삭제(16파일)** — 중복 구현 | — |

## 두 404 신호 (별개, 둘 다 살아 있음)
- **GW 404**: `🚧 [GW 404] 라우트 없음 …` + `Request/Host/Service/Time`. (Host, 정규화 경로) 쿨다운 60s + 60건/5분 상한.
- **앱 404**: `🔎 [404] 매핑 없음 — 이관 누락 후보` + `Path/Service/Time`. dedupe 키 `404:/…/{}`, `RenewSlackNotifier` 쿨다운 10s.
- 공통: **메시지엔 원경로 그대로**(정규화는 키에만), 노이즈(favicon·robots·sitemap·.well-known·확장자·`/wp-`·`/.env`·`/.git`·`/vendor/`·`/cgi-bin`·`/console`) 제외.

## 테스트 결과
`./gradlew cleanTest test` 4개 전부 그린 — booster **535**, reviewed **239**, lawkit **182**, gateway **76** (failures 0 / skipped 0). 중복 구현 삭제로 이전 보고(547/254/197/84)보다 줄었다.
**실발송 실증(가짜 웹훅 :19911)** — 실제 수신 텍스트:
```
🚧 [GW 404] 라우트 없음 — 게이트웨이 라우트표에 매칭이 없다     🔎 [404] 매핑 없음 — 이관 누락 후보
Request: GET /hospital/apis/x/25                              Path: GET hospital/apis/hospital/unrestricted/detail/8812
Host: unknown-25.example.com                                  Service: booster-api
Service: gateway                                              Time: 2026-08-09T13:05:42.733233
Time: 2026-08-09T13:02:36.706090
```
GW: dev 프로파일 기동(`slack error-notify: enabled`) → id 만 다른 404 3건 → **1건**, 노이즈 3건(favicon/robots/.env) → **0건**, 업스트림 다운 → 별도 `💥 [ERROR] … 게이트웨이 upstream / Detail: route=user target=… `.
앱: advice→RenewSlackNotifier→웹훅 실경로로 id 3건 → **1건**, 노이즈 2건 → **0건**.

## 알려진 한계 / 리뷰어에게
- **이모지 1자 확인 필요**: 원본 소스와 실제 발송본은 `🔎`(U+1F50E)인데 전달받은 사용자 샘플엔 `🔍`(U+1F50D)로 적혀 있었다. 커밋된 소스를 신뢰해 `🔎` 로 맞췄다(위 실수신 텍스트도 `🔎`). 사용자 화면이 `🔍` 이면 1자만 고치면 된다.
- **`Service:` 라벨**: 원본 규약이 `{service}-api` 라 404 알림은 `booster-api`/`reviewed-api`(사용자 샘플과 일치)/`lawkit-api` 로 뒀다. reviewed 의 **예외** 알림은 기존 `APP_LABEL="reviewed-app"` 을 그대로 둬서 404 와 라벨이 다르다 — 둘 다 사용자가 지금 보고 있는 값이라 손대지 않았다. 통일할지 판단 필요.
- **게이트웨이 패키지**: 새로 만든 예외 알림 2개를 `gateway.notify` → `gateway.alert` 로 옮겨 `RouteMissAlerter` 와 같은 패키지에 뒀다. 원본의 package-private `normalizeForDedupe` 를 public 으로 넓히지 않고 재사용하기 위해서다(원본 무수정 유지).
- **미구현**: 게이트웨이 `SlackBootFailureListener`(부팅 실패 알림). 앱 3종엔 있고 게이트웨이엔 없다 — 지시 범위 밖이라 손대지 않았다.
- 4벌 중복 문제는 사라졌다. 앱 3종은 기존 `RenewSlackNotifier` 를 그대로 쓰고 **새 클래스가 0개**다.

STATUS: REVISED
