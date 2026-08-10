# 변경 요약: P0 기반·인증 (R1 `modules/legacy` 신설 + R2 인증경계 이설)

브랜치 `feature/prod-sync-candidates` @ `/Users/steve/steve/legalcare-renew-prodsync-wt` · **커밋 0건**
선행 조건 ①(기존 테이블 5개 운영 정합)은 그린 — 근거는 `P-1-schema-sync.md` 19절.

> **라운드1 반영 후 최신본이다.** 라운드1 리뷰(`round-1-P0-review.md`)의 CRITICAL 1 + MAJOR 다수를
> 반영해 이 문서를 통째로 정정했다. 무엇을 어떻게 고쳤는지는 `round-1-P0-response.md`.

## 변경 파일
| 파일 | 한 줄 요약 | R# |
|---|---|---|
| `apps/booster-app/settings.gradle` | 10번째 모듈 `legacy` 등록 | R1 |
| `apps/booster-app/modules/legacy/build.gradle` | 신규 모듈 빌드 정의(`:core` + web + lombok) | R1 |
| `apps/booster-app/modules/legacy/.../legacy/health/HealthCheck.java` | 구 `HealthCheck.kt` 1:1 이식 (`GET /api/v1/health`, 200·본문 0바이트) | R1 |
| `apps/booster-app/app/build.gradle` | `:app` 이 `:modules:legacy` 의존 + 토큰 프로브용 `jjwt-api` 테스트 의존 | R1·R2 |
| `apps/booster-app/modules/legacy/src/test/.../HealthCheckTest.java` | 헬스체크 와이어 계약(200·빈본문·프리픽스 없음) 고정 | R1 |
| `apps/booster-app/app/src/test/.../LegacyModuleMountTest.java` | 실조립 컨텍스트: `/api/v1/health` 리터럴 등록 + 정책 빈 등록 + **추출기 주입 순서** | R1·R2 |
| `apps/booster-app/core/.../core/filter/RestrictedPathPolicy.java` | 경로별 인증경계 확장점(신규 인터페이스) + `rejectsInvalidToken()` | R2 |
| `apps/booster-app/core/.../core/filter/TokenVerificationFilter.java` | 소유권 주장 정책이 있으면 그 판정(+잘못된 토큰 거부·리다이렉트 금지), 없으면 기존 `bypass.url` 판정(폴백) | R2 |
| `apps/booster-app/app/src/main/.../security/LegacyPathAuthPolicy.java` | 구 `SecurityConfig.kt:36-62` permitAll 26규칙을 선언 순서 그대로 이식 | R2 |
| `apps/booster-app/app/src/main/.../security/LegacyHeaderTokenExtractor.java` | **`/api/v1` 토큰을 `Authorization` 헤더에서 읽는다** (구 `JwtAuthenticationFilter.kt:39`) | R2 |
| `apps/booster-app/app/src/test/.../security/LegacyPathAuthPolicyTest.java` | 115경로 전수 + 소유범위 + 경계조건 (132 케이스) | R2 |
| `apps/booster-app/app/src/test/.../security/LegacyTokenSourceTest.java` | **토큰을 실어서** 출처·만료처리·무회귀·`resolveRestricted` 고정 (16 케이스) | R2 |
| `apps/booster-app/app/src/test/resources/legacy-auth-boundary.tsv` | 위 테스트의 기대표(구 파일:라인 출처 포함) — 추출기 정정 후 재생성 | R2 |
| `apps/booster-app/app/src/main/resources/db/legacy-absorption-p1-schema.sql` | 신규 PG 부족분 1차 DDL(멱등) | R11 |
| `apps/booster-app/app/src/main/resources/db/legacy-absorption-p1b-schema.sql` | 운영 기준 42테이블 동기화 DDL(멱등, D절은 단일 트랜잭션) | R11 |
| `apps/booster-app/app/src/main/resources/db/legacy-absorption-p1c-existing-align.sql` | 작업① 기존 테이블 5개 운영 정합 DDL(멱등) | R11 |

**16파일 = `git status --short` 실측치와 일치**(수정 3 + 신규 13, `modules/legacy/**` 3파일 포함).
`build/` 산출물은 `apps/booster-app/.gitignore:2` 로 제외. R# 에 매핑되지 않는 변경 파일 **없음**.
게이트웨이(`Routes.java`·`RouteTableTest`) 무수정 — `Routes.java:34` 는 여전히 `ServiceId.LEGACY`.
`medilawyer-boot` 무수정.

## 테스트 결과
```
$ ./gradlew test --rerun-tasks          (apps/booster-app, 전체 스위트)
BUILD SUCCESSFUL in 4m 25s
  app 195 · core 48 · legacy 2 · chrono 7 · crawling 68 · external 40 · medicontents 18
  · notification 87 · post 67 · product 71 · user 86
  TOTAL tests=689 failures=0 errors=0 skipped=0    (P0 착수 전 667 → 초안 672 → 라운드1 후 689)

$ ./gradlew duplicateClassCheck         BUILD SUCCESSFUL (클래스 중복 0)

$ ./gradlew test --rerun-tasks          (apps/gateway-app)
BUILD SUCCESSFUL — TOTAL tests=76 failures=0, 그중 RouteTableTest 22개 통과
  Routes.java:34 `new Route("legacy", MEDILAWYER, "/api/v1", ServiceId.LEGACY)` 그대로
```
DDL: p1b 를 D절 트랜잭션 래핑 후 **2회 재적용 exit=0 · ERROR 0 · WARNING 0**, 신규 PG BASE TABLE 144 불변.

## AC 자가 점검
**AC1** ✅ ①`:app:test` 그린(195/0) ②기동 로그에 매핑 노출 — `bootRun` 실제 기동 로그 원문:
`TRACE ... RequestMappingHandlerMapping : l.m.l.h.HealthCheck: {GET [/api/v1/health]}: checkHealth()`
(`/actuator/mappings` 도 `{GET [/api/v1/health]}`, `/actuator/surface` 도 `pattern=/api/v1/health`)
③게이트웨이는 여전히 legacy-bridge 로 감 — `RouteTableTest` 무변경·22개 통과, 라우트 목적지 `LEGACY` 유지.

**AC2** ✅ ①대조표 `P0-auth-boundary.tsv` **130행 × 16열, 공란 0**(생성 시 단언).
판정 일치 **116** / 의도적 불일치 **14**(`/internal`) = **130** — 130행 전부 계상한다(라운드1 지적6 정정).
②실측 — **무토큰만이 아니라 유효토큰까지** 쟀다. 무토큰 프로브는 "게이트가 막았다"와 "추출기가 헤더를
안 읽었다"를 구분하지 못한다는 것이 라운드1 CRITICAL 로 드러났기 때문이다.

| 구간 | 행 | 구 무토큰 | 신 무토큰 | 신 유효토큰(수정 후) | 신 유효토큰(수정 전) |
|---|---|---|---|---|---|
| `/api/v1` AUTHENTICATED | 66 | **401** | **401** | **404**(게이트 통과) | **401** ← 회귀분 |
| `/api/v1` PERMIT_ALL(health 제외) | 48 | 미실측 | 404 | 404 | 404 |
| `GET /api/v1/health` | 1 | **200** | **200** | **200** | 200 |
| `/api/v2` (소유 범위 밖) | 1 | **401** | **401** | 401(쿠키 출처 유지) | 401 |
| `/internal` (P4 이월) | 14 | 미실측 | 403 | 403 | 403 |

**무토큰 열은 수정 전후가 완전히 같다**(66×401). 라운드1 리뷰가 "66건 일치는 공허하다"고 한 것이 그대로
재현됐고, 같은 유효 토큰을 실은 열에서만 66행이 401→404 로 뒤집힌다. 신 쪽은 130/130 전수 실측이다.

구 쪽 유효토큰은 **표본 3건**만 쟀다 — 유효 토큰을 실으면 구는 컨트롤러를 실제로 실행하므로
(카카오·네이버·알림톡·기프트, 쓰기 메서드는 DB 변경까지) 불변조건 5 를 지키려면 전수는 불가능하다.
표본은 순수 조회임을 소스로 확인하고 골랐다: `GET /api/v1/app-users/me` 200 · `GET /api/v1/users/me/stores` 200 ·
`GET /api/v1/health` 200. **같은 토큰을 쿠키로만 보내면 구는 401** — 구가 헤더만 읽는다는 성질의 직접 증거다.
permitAll 48개에 대한 게이트 통과 실증은 초안대로 **핸들러 없는 하위경로 프로브**가 담당한다
(프로브 창 3,248줄에 `*Controller/*Service` 로그 0건, 외부 송신 0건).

## 알려진 한계 / 리뷰어에게

**1. `/api/v1` 밖은 일부러 안 건드렸다.** `LegacyPathAuthPolicy.owns()` 는 `/api/v1` 하위만 주장한다.
`/api/v2` 는 구·신 판정이 이미 둘 다 "토큰 필요"로 같아서(실측 401) 손댈 이유가 없고,
`/internal/**` 14경로는 `InternalAccessBlockFilter`(T14, HIGHEST_PRECEDENCE)가 인증 이전에 403 을 내므로
여기에 규칙을 심어도 한 줄도 실행되지 않는다. 구에서는 이 14개가 전부 permitAll 이었으니 **동작은 다르다** —
대조표에 "의도적 불일치(P4)"로 남겼다. 실제 처분은 컨트롤러가 들어오는 P4 의 몫이다.

**2. 구 `ApiV1Filter` 는 옮기지 않았다.** 구는 `/api/v1`·`/api/v2`·`/internal` 밖 요청에 403 "Access Denied"
를 준다. 이건 "이 *서비스*는 세 접두사만 서빙한다"는 선언이지 경로별 인증 규칙이 아니고, 9모듈이 한 디스패처를
공유하는 booster 에 넣으면 `/user/**`·`/post/**` 등 전 모듈이 403 이 된다. 실측으로도 booster 는
`/nope/nothing` 에 401(구는 403)을 낸다. 게이트웨이 legacy 라우트는 `/api/v1` 만 넘기므로 P6 이후에도
이 차이가 노출되는 경로는 없다고 보지만, **"구와 100%"의 예외로 명시해 둔다.**

**3. 에러 봉투가 다르다 — P6 전에 결정해야 한다(라운드1 리뷰 정정 반영).**
```
무토큰   구 401 {"errors":[{"code":"AUTH-001",...}],"errorMessage":null,"path":"uri=...","timeStamp":"..."}
        신 401 {"status":401,"code":"90008","message":"토큰이 필요합니다."}
만료토큰 구 400 {"errors":[{"code":"AUTH-006",...}]}          <- 상태코드까지 다르다
        신 401 {"status":401,"code":"90006","message":"토큰이 만료됐습니다."}
```
**리플레이는 이걸 영영 못 잡는다.** 판정 기준이 "원래 에러가 나던 요청이 그대로 에러면 통과"라 401↔401 도
400↔401 도 통과시킨다. 즉 P5 를 그린으로 통과한 뒤 **전환 당일 클라이언트에서 처음 드러난다.**
그래서 시한을 P5 가 아니라 **P6 이전**으로 옮긴다. 지금 안 고친 이유는 이 봉투가
`TokenVerificationFilter` 의 공용 catch(상태 401 고정)에서 나와 기존 9모듈 전체가 같이 움직이기 때문이다.
결정 사항: ①`errors[0].code` 를 파싱하는 클라이언트가 실재하는지 ②만료 토큰의 400 vs 401 을 맞출지.

**4. 핸들러 없는 경로의 상태코드가 다르다 — 구 500, booster 404. 방향이 위험하다(라운드1 리뷰 정정 반영).**
구는 `NoResourceFoundException` 을 `GlobalExceptionHandler` 가 `SERVER-001` 500 으로 처리하고
(`{"errors":[{"code":"SERVER-001","message":"No static resource ..."}]}`) booster 는 404(`{"code":"40400",...}`).
초안은 이걸 "조용히 넘어간다"고 썼는데 **정확히는 판정 기준이 통과시켜 버린다** — "원래 에러가 나던 요청이
그대로 에러면 통과"이므로 500→404 는 합격으로 찍힌다. 그리고 이 시나리오는 가정이 아니다:
라운드1 에서 실제로 `GET /api/v1/team/presentHistories/{teamId}/excel` 이 추출기 결함으로 대조표에서
빠져 있었고, 그대로 갔으면 이식 누락분이 리플레이에서 합격으로 찍혔을 것이다. → 아래 P5 판정 규칙 추가.

## P5 판정 규칙 추가분 (P0 에서 확정 — P5 착수 시 하네스에 반영할 것)
기준 문서는 `검증방식-리플레이-차등검증.md` 이고 지금 규칙은 *"200이 나와야 할 요청이 다른 에러로 바뀌면
불합격, 원래 나던 에러는 그대로면 통과"* 하나다. 그 규칙이 통과시켜 버리는 것을 아래로 보강한다.
그 문서는 다른 태스크가 공유하는 절차 원본이라 이번에 직접 고치지 않았다 — **P5 착수 시 이 절을 그 문서에
반영하는 것이 착수 조건이다.**

1. **`구 500 → 신 404` 는 불합격**으로 판정한다. 에러↔에러라도 이 조합은 "이식 누락"의 고유 서명이다
   (구는 핸들러 없는 경로에 500, 신은 404).
2. **바디 비교를 켠다.** 상태코드만 보면 permitAll 경로의 신원 유무를 구분할 수 없다(아래 6번).
3. **진입점을 게이트웨이로 고정한다.** booster 에 직결하면 골든셋 `/api/v1` 369건 중 `OPTIONS` 181건이
   전부 401 로 찍혀 회귀로 오독된다(CORS 는 엣지 담당 — 구 `CustomCorsFilter.kt` 전체 주석,
   신 `GatewayConfig.java:58-76` 의 `CorsFilter`).
4. 에러 봉투 차이(3번)는 리플레이 판정 대상이 아니다 — **P6 전 결정 항목**으로 별도 관리한다.

**5. `TokenVerificationFilter` 를 건드린 것이 이번 P0 에서 가장 위험한 변경이다.**
9모듈 전 경로가 이 필터를 탄다. 그래서 기존 판정 경로를 고치지 않고 **정책이 소유를 주장할 때만 분기**하도록
덧붙였고, 폴백은 종전 `requestHandler.useToken(tokenBypassUrl)` 그대로다. `List<RestrictedPathPolicy>` 가 아니라
`ObjectProvider` 로 받은 것도 같은 이유다(후보 빈 0개일 때 생성자 주입이 부팅을 깨뜨린다).
같은 원칙을 **토큰 출처**에도 적용했다 — `ApiPathMatcher` 를 고치지 않고 `/api/v1` 만 주장하는 추출기를
하나 더 얹었다. 무회귀는 실측으로 확인했다: `/user/apis/unrestricted/x`→404, `/external/apis/swagger`→404,
`/post/apis/x`→401, `/api/v2/boost1/...`→401(헤더 토큰을 실어도 401, 쿠키 토큰이면 404) — 전부 변경 전과 같다.

**6. permitAll 경로의 신원 — 지금은 채워지고, 실제로 바디를 가르는 경로가 있다.**
라운드1 지적대로 수정 전에는 `/api/v1` 이 쿠키를 읽어 permitAll 경로가 200 이면서 **익명**으로 돌았다.
헤더를 읽게 고쳐 `appUserId`/`email`/`organizationRole` 이 심긴다. 영향 범위를 구 소스로 전수 확인했다 —
permitAll 49경로 중 25개는 `@AuthenticationPrincipal` 자체가 없고, 20개는 선언만 하고 본문에서 안 쓰며,
**4개가 실제로 쓴다**:
| 경로 | 신원이 없으면 |
|---|---|
| `POST /api/v1/organizations/{organizationId}/invitations` | `AUTH_REQUIRED` 로 **400** (`OrganizationController.kt:249`) — permitAll 인데 신원 필수 |
| `POST /api/v1/store/register` | `appUser?.id` 가 크롤 요청 Kafka 메시지에 실린다 → **write-set 이 달라진다** (`StoreFacade:100`) |
| `GET /api/v1/posts` | `appUser` 를 넘기지만 쿼리·DTO 모두 미사용 → 현재 코드에서는 응답 동일 |
| `GET /api/v1/stores/{storeId}/posts/statistics` | `appUserId` 를 넘기지만 `PostRepositoryImpl:125` 이 미사용 → 응답 동일 |

**7. 로컬 기동은 `SPRING_PROFILES_ACTIVE=local` 이 필요하다.** `-Dspring.profiles.active=local` 은 bootRun 이
포크하는 JVM 에 전달되지 않아 `spring.redis.data.host` 미해석으로 부팅이 깨진다(내 변경과 무관한 기존 함정).
`/actuator/health` 는 로컬에 DB 가 없어 응답하지 않는다 — `/actuator` 자체는 200.

**8. 미판정.** ①구 `SecurityConfig` 규칙을 읽어 만든 기대표와 Java 구현이 서로를 검증하는 구조라 **둘 다 같은
오독을 하면 잡히지 않는다.** 실측 67경로 일치 + 유효토큰 전수 스윕으로 상당 부분 걷어냈지만
permitAll 48경로의 구 실측은 여전히 없다(사유는 AC2). P1~P4 에서 컨트롤러가 들어와 리플레이를 태울 때 해소된다.
②구·신이 **같은 JWT 서명키를 쓰는지 확인하지 못했다.** 구는 config-server+Vault 에서 키를 받고
`/legacy-service/workstation,git,vault` 조회는 `propertySources: []` 로 비어 돌아온다. 그래서 이번 실측은
구·신 각각 자기 키로 서명한 토큰을 썼다 — "각 서버가 헤더를 읽는가"는 증명되지만 "구 토큰을 신이 검증하는가"는
아니다. 그건 원래 **AC5(P3)** 의 4조합 항목이다. P3 착수 시 키 소재를 먼저 확정할 것.
③`/api/v1` 무토큰 응답이 `token.fail.redirect-url` 값에 따라 302 로 바뀔 수 있던 함정은 이번에 닫았지만
(`TokenVerificationFilter` 의 `owningPolicy != null` 분기 + 테스트), `docs/audit/F-config-wiring.md:19,53` 의
🟠 항목 자체는 그대로다. 그 값을 채우는 사람은 `/api/v1` 이 예외라는 것을 알아야 한다.

STATUS: DONE
