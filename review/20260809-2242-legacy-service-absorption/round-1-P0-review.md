# Round 1 — P0(기반·인증) 교차 리뷰

수정 권한 `review` — 코드는 한 줄도 건드리지 않았다. 근거는 `git diff HEAD` + 신규 파일 직접 읽기 +
구 `medilawyer-boot@930c83cf` 원본 + 실행 가능한 것은 직접 실행한 결과다.

먼저 잘 된 것부터. **26규칙 이식 자체는 정확하다.** `SecurityConfig.kt:36-62` 를 직접 펴서 한 줄씩 대조했고
`LegacyPathAuthPolicy.java:78-103` 과 (메서드 × 패턴 × 선언순서)가 26/26 일치한다. `:47`·`:63` 이 빈 줄이라
주석의 라인번호가 건너뛴 것도 원본대로다. 매칭 엔진 선택도 검증했다 — `PathPatternParser` 가 구
Spring 6.1.13(Boot 3.3.4)과 신 7.0.8(Boot 4.0.7)에서 다른 답을 내지 않는지 의심스러워서, 26규칙을 그대로 담은
프로브를 **두 버전 jar 로 각각 컴파일해 44개 경계 케이스를 돌렸다. 출력 차이 0** (트레일링 슬래시·`*` 세그먼트·
퍼센트 인코딩·matrix·중복 슬래시·대소문자 포함). 버전 격차는 위험 요소가 아니다.
폴백 안전성도 확인했다 — `owns()` 는 `/api/v1` 접두사만 주장하고, booster 전 모듈의 `@RequestMapping` 을
훑어 `/api/v1` 을 서빙하는 기존 컨트롤러가 **하나도 없음**을 확인했으므로(`/apis`·`/internal`·`/api/v2` 뿐)
기존 9모듈 수백 경로는 전부 `requestHandler.useToken()` 으로 그대로 떨어진다. 판정이 바뀌는 기존 경로는 없다.

그런데 **이설된 것은 인증 경계의 절반뿐이다.** "어떤 경로가 토큰을 요구하는가"는 옮겼는데
"그 토큰을 어디서 읽는가"는 안 옮겼다. 아래 1번이 이 리뷰의 전부라고 봐도 된다.

---

## 1. CRITICAL — `/api/v1` 은 `Authorization` 헤더가 아니라 **쿠키**에서 토큰을 찾는다

`TokenVerificationFilter.java:72` 가 `selectExtractor()` 로 고른 추출기가 토큰을 읽는데, 그 선택이
`ApiPathMatcher.isApiPath()` 하나로 갈린다.

- `HeaderTokenExtractor.java:19` → `supports() = ApiPathMatcher.isApiPath(uri)`
- `CookieTokenExtractor.java:23` → `supports() = !ApiPathMatcher.isApiPath(uri)`, 그리고 `:29` 에서
  **쿠키 `accessToken`** 을 읽는다
- `ApiPathMatcher.java:23-27` 은 `/apis` 와 `/{module}/apis` 만 참으로 본다

`isApiPath("/api/v1/...")` 를 그 메서드 원문 그대로 떼어 실행해 확인했다 — **false**. 즉 `/api/v1` 전 경로가
`CookieTokenExtractor` 를 타고, 구가 읽던 `Authorization` 헤더(`JwtAuthenticationFilter.kt:39`)는
**한 번도 읽히지 않는다.** booster 에 `TokenExtractor` 구현은 이 둘뿐이다(grep 전수).

골든셋으로 실측했다(DELL 읽기 전용, 35파일 118,233엔트리 — 토큰 값은 출력하지 않고 헤더 **이름**만 집계):

| `/api/v1` 캡처 369건 | 건수 | 그중 캡처 응답 200 |
|---|---|---|
| 토큰필요 경로 + `Authorization` 헤더 보유 | **92** | **87** |
| permitAll 경로 + `Authorization` 헤더 보유 | 60 | 56 |
| `OPTIONS` 프리플라이트(토큰 없음) | 181 | 181 |

**쿠키를 단 하나라도 실은 `/api/v1` 요청은 0건이다**(`cookie` 헤더 자체가 없다). 그러니 위 92건은
booster 에서 `token` 이 공백 → `isRestrictedUrl=true` → `TokenVerificationFilter.java:106` 의
**401 `90008`** 로 떨어진다. 캡처는 87건이 200이었다.

판정 기준(`검증방식-리플레이-차등검증.md:88`)은 *"200이 나와야 할 요청이 다른 에러로 바뀌면 불합격.
이게 유일한 실패 조건이다"* 다. 정확히 그 조건이다. P5 가 아니라 **P1 리플레이에서 바로 터진다.**
게이트웨이는 `Authorization` 을 그대로 넘긴다(`ProxyFilter.java:49,53` hop-by-hop 목록에 `authorization`
없음)이므로 중간에서 구제되지도 않는다.

**AC2 가 이걸 왜 못 봤는지가 핵심이다.** 66경로를 전부 *무토큰*으로 찔렀다. 게이트가 정상이어도 401,
추출기가 헤더를 아예 못 읽어도 401이다. **401/401 66건은 두 경우를 구분하지 못한다.**
오케스트레이터가 물은 "둘 다 제대로 막았나, 둘 다 엉뚱한 이유로 실패했나"의 답은 — 구분 안 되고,
실제로 후자다. 유효 토큰 1개로 `GET /api/v1/users/me/stores` 를 양쪽에 찔렀으면 P0 에서 즉시 잡혔다.

`ApiPathMatcher` 를 고치면 `/apis` 계열 기존 판정이 흔들리니 그쪽은 건드리지 말고,
`RestrictedPathPolicy` 를 만든 것과 같은 방식으로 **추출기 선택에도 소유권 개념을 넣는 것**이 이 변경의
설계와 일관된다. 어느 쪽이든 P1 착수 전에 닫아야 한다.

## 2. MAJOR — permitAll 49경로는 200을 내면서 **조용히 비회원으로** 동작한다

같은 뿌리인데 실패 모양이 다르고, 탐지 방법도 달라서 따로 적는다. 1번이 고쳐지기 전까지
permitAll 경로에서는 `TokenVerificationFilter.java:79-81` 의 `appUserId`/`email`/`organizationRole`
요청 속성이 **하나도 안 심긴다.** 구는 같은 요청에서 `SecurityContextHolder` 에 인증을 채운다
(`JwtAuthenticationFilter.kt:75-81`).

`SecurityConfig.kt:48` 주석이 `GET /api/v1/posts/**` 를 *"리뷰 리스트 조회 (회원용 + 비회원용)"* 이라고
못 박아 뒀다 — 응답 내용이 신원에 따라 갈리는 경로다. 골든셋에 이 모양이 60건(200 기대 56건) 있다.
**상태코드는 200/200 으로 일치하므로 상태코드만 보는 대조는 전부 통과시킨다.** AC3 의
"구 200 → 신 비200 0건" 규칙으로는 원리적으로 못 잡는다. P1 판정에 바디 비교가 반드시 필요하다.

## 3. MAJOR — 구는 permitAll 경로에서도 **만료 토큰을 거부**하는데 booster 는 무시하고 통과시킨다

구: `JwtAuthenticationFilter` 는 인가 판정보다 먼저 전 경로에서 돈다. `JwtProvider.kt:109-111` 이
만료 시 `CustomException(AUTH_EXPIRED_TOKEN)` 을 던지고 → `JwtAuthenticationFilter.kt:94-97` 이 받아
`resolver.resolveException` → `GlobalExceptionHandler.kt:38` 이 **400** 을 낸다. `filterChain.doFilter` 는
호출되지 않아 컨트롤러가 안 돈다.

신: `TokenVerificationFilter.java:95-102` 는 `!isRestrictedUrl` 이면 warn 로그만 남기고 **그대로 진행**한다 → 200.

이 분기는 종전에는 `/api/v1` 에서 한 번도 안 돌았다(전 경로가 restricted 였으므로). **49경로를
unrestricted 로 만든 이번 변경이 이 분기를 새로 켠 것**이라 P0 의 책임 범위다. 방향이 `구 400 → 신 200`
이라 AC3 의 단방향 규칙에 또 안 걸린다. 지금은 1번에 가려 보이지 않지만(헤더를 아예 안 읽으니 만료 토큰을
만날 일이 없다) **1번을 고치는 순간 살아난다. 같이 처리해야 한다.**

## 4. MAJOR — `p1b-schema.sql:3643` 의 `DROP TABLE` 이 트랜잭션 밖이다

```
3643      DROP TABLE public.boost_message_delivery;      -- 재생성은 3647
```
가드는 3중이고(테이블 존재 → 컬럼 순서가 운영과 다름 → **행수 0**) 자동 실행 경로도 없다
(flyway·liquibase 없음, `spring.sql.init` 없음, `ddl-auto: none`, 이 파일들을 읽는 코드·테스트 0건).
그런데 **p1b 3,751줄 전체에 `BEGIN;`/`COMMIT;` 이 하나도 없다.** 3645~3672 사이에서 죽으면 테이블이
사라진 채로 남는다(비어 있더라도 앱은 쓰기에서 깨진다). 동기가 "물리 컬럼 순서 맞추기"라는 점에서도
`DROP TABLE` 은 명시적 승인이 필요한 종류의 변경이다. 섹션 D 를 트랜잭션으로 감쌀 것.

참고로 **P-1c 는 깨끗하다.** 승인 범위 5개 테이블(`boost_post_similarity`, `similarity_score_analysis_data`,
`boost_present_order`, `boost_message`, `post`) 정확히 그것만 건드리고 그 밖은 0건. 전 문장 가드 있고
파괴적 구문 없다. `p1c:49` 의 타입 가드가 `IS DISTINCT FROM 'varchar(255)'` 라 넓히기가 아니라 *다르면*
실행이라는 점만 짚어 둔다(현재 컬럼이 `text`/`varchar(500)` 이면 축소 시도 → PG 가 에러로 중단, 조용한
손실은 없음. `atttypmod` 비교로 좁히면 깔끔).

## 5. MAJOR — 변경 파일 표가 실제 워킹트리와 다르다 (04-changes-P0.md:20,22)

`git status --short` 에 `legacy-absorption-p1-schema.sql` 과 `legacy-absorption-p1b-schema.sql` 이
untracked 로 있는데 표에는 `p1c` 한 줄뿐이고, `:22` 는 *"R# 에 매핑되지 않는 변경 파일 없음"* 이라고
단언한다. 두 파일 다 R11(P-1) 산출물이 맞으니 **고아 변경은 아니다.** 하지만 목록도 단언도 사실과 다르다.
리뷰어가 `git diff` 로 확인하는 문서에서 인벤토리가 틀리면 그 문서의 다른 단언도 같이 못 믿게 된다.
(모듈 `build/` 산출물은 `apps/booster-app/.gitignore:2` 로 제외됨 — 확인함, 문제 없음.)

## 6. MAJOR — 대조표가 스스로와 모순되고, AC2 표가 15행을 누락한다

- `P0-auth-boundary.tsv:117-131` 15행: 12열이 `부분검증: booster 측 실측만` 인데 같은 행 11열은 `미실측` 이다.
  실측 안 한 행에 "실측만 했다"는 판정이 붙어 있다.
- `04-changes-P0.md:63,95` 는 `/api/v2/boost1/...` 를 **"실측 401"** 이라고 두 번 쓰는데,
  TSV `:117` 은 그 행을 구·신 **양쪽 미실측**으로 기록한다. 둘 중 하나는 틀렸다.
- AC2 표(`:48-52`)는 66+1+48=**115행**만 계상해 130행 표의 15행이 사라졌고, AUTHENTICATED 는 67행인데
  66을 **"전수"** 라고 적었다. 빠진 1행이 하필 `:117` 이다.
- "공란 0"은 문자 그대로는 참이나, 측정 셀 260칸 중 **78칸이 상태코드가 아니라 산문**이다
  (구 실측 63행 `미실측: permitAll…`, booster 실측 15행 `미실측: booster 미서빙…`).
  AC2 의 자구는 만족하지만 증거로서의 무게는 그만큼 아니다.

## 7. MAJOR — 실제 엔드포인트 1개가 대조표·테스트에서 통째로 빠졌다

`GET /api/v1/team/presentHistories/{teamId}/excel` (`TeamController.kt:70-73`) 이 P0 대조표에도
테스트 픽스처에도 없다. 대신 `P-1-endpoint-table-map.tsv` 스스로 `엔드포인트 아님`으로 분류한
`GET /api/v1/team` 이 그 자리에 있다(TSV `:109`, 픽스처 `:111` — 경로는 클래스 레벨인데 출처는 `kt:70` 을
가리키는 게 증거다). 원인은 추출기가 **여러 줄에 걸친 `@GetMapping(value = [...])`** 를 못 읽고 클래스 레벨
`@RequestMapping` 으로 폴백한 것.

인증 판정 결론은 안 바뀐다(둘 다 `GET /api/v1/team/**` → permitAll). 문제는 두 가지다.
①`LegacyPathAuthPolicyTest` 의 "115경로 전수"는 실은 **114 실경로 + 1 유령**이고 실경로 1개는 무검증이다.
②같은 모양의 애노테이션이 다른 컨트롤러에 또 있으면 똑같이 새어 나간다 — **P1 착수 전 전 컨트롤러 스윕**이 필요하다.
작성자 스스로 4번 항목에서 "이식 누락은 조용히 넘어간다"고 썼는데, 그 예시가 이미 발생해 있다.

## 8. MINOR — `/api/v1` 이 "API 경로가 아님"으로 분류되어 **브라우저 리다이렉트 분기**에 얹혀 있다

같은 `ApiPathMatcher` 가 `TokenVerificationFilter.java:105` 에서도 쓰인다. `/api/v1` 이 false 이므로
무토큰 restricted 요청은 `:110-113` 의 `sendRedirect` 분기를 먼저 만난다. 지금 401 이 나오는 건
`token.fail.redirect-url` 이 비어 있어서일 뿐이다(booster yml 5개 전부에 없음 — 확인함).
그런데 `docs/audit/F-config-wiring.md:19,53` 이 이 값 부재를 🟠 미해결로 올려 두었고 구 값은 `/user/sign-in`
이다. 누가 그 감사 항목을 "해결"하는 순간 **무토큰 `/api/v1` 전건이 401 → 302 로 조용히 바뀐다.**
못박는 테스트도 주석도 없다.

## 9. MINOR — `resolveRestricted()` 자체에 테스트가 없다

`TokenVerificationFilter.java:143-151` 은 10개 모듈 전 요청이 타는 자리인데, ①정책이 소유하면 `bypass.url`
판정이 안 도는지 ②정책이 없을 때 종전과 완전히 같은지 ③javadoc 이 규약으로 못 박은 다중 정책 `@Order`
타이브레이크 — 셋 다 테스트가 없다. 기존 9모듈 무회귀 근거가 수동 curl 4건(`04-changes-P0.md:94-95`)뿐이다.
정책 쪽에 132케이스를 깐 것에 비하면 위험도 대비 배분이 거꾸로다.

## 10. MINOR — 프리플라이트는 결함 아님, 다만 P5 하네스 설정에 못박을 것

골든셋 `/api/v1` 369건 중 **181건이 `OPTIONS` 이고 전부 200** 이다. `LegacyPathAuthPolicy` 에 OPTIONS 규칙이
없어 `isRestricted("OPTIONS", …)` 는 전부 restricted 를 낸다. **하지만 제품 결함은 아니다** — 구는
`CustomCorsFilter.kt` 가 통째로 주석이고 구 게이트웨이 globalcors 가 엣지에서 처리했으며, 신도
`GatewayConfig.java:58-76` 의 `CorsFilter`(HIGHEST_PRECEDENCE)가 같은 자리를 맡는다. 게이트웨이로 들어오면
booster 까지 오지 않는다. 다만 **리플레이 샤드를 booster 에 직결하면 181건이 전부 401 로 찍혀 회귀로 오독된다.**
진입점을 게이트웨이로 고정하는 것을 P5 설정에 명시할 것.

## 11. MINOR — 죽은 규칙과 과장된 주석 하나씩

`SecurityConfig.kt:49 GET /api/v1/posts/statistics/**` 는 한 줄 앞 `:48 GET /api/v1/posts/**` 에 완전히
가려진 **구에서도 죽은 규칙**이다. 이식은 충실하니 고칠 것 없고, 나중에 누가 "정리"하지 않도록 적어 둔다.
같은 맥락에서 `LegacyPathAuthPolicy.java:15-16` 의 *"순서를 바꾸면 판정이 바뀐다"* 는 이 규칙집합에는
성립하지 않는다 — 26개가 전부 permitAll 이고 폴백이 authenticated 라 순서와 무관하게 답이 같다.
순서 보존은 여전히 옳은 선택이지만, 테스트가 지키는 불변식은 주석이 주장하는 것보다 약하다.

---

## 작성자 자기신고 4건 판정 (오케스트레이터 질문 3)

| 신고 | 심각도 판정 | 근거 |
|---|---|---|
| `/internal` 14경로 구 permitAll vs booster 403 | **동의(MINOR)**. P4 이월 타당 | `Routes.java` TABLE 에 `/internal` 라우트가 아예 없다 → 외부 노출 경로 없음. 단 `InternalAccessBlockFilter:174` 가 `internal` 세그먼트를 **무조건** 403 하므로 P4 는 컨트롤러만 옮겨선 안 되고 예외구멍 설계가 따로 필요하다 — 그 전제가 문서에 없다 |
| 401 바디 `AUTH-001` vs `90008` | **동의(MINOR)**, 단 시한은 P5 가 아니라 **P6 이전** | 판정 기준이 기능 중심("원래 에러면 통과")이라 리플레이는 통과시킨다. 즉 리플레이로는 영영 안 잡히고 전환 당일 클라이언트에서 드러난다 |
| 핸들러 없는 경로 구 500 / 신 404 | **동의(MINOR)**, 다만 이유가 반대 | 작성자는 "조용히 넘어간다"고 썼는데, 정확히는 판정 기준이 *"원래 에러가 나던 요청이 그대로 에러면 통과"* 라 **리플레이가 통과시켜 버린다.** 7번이 그 첫 사례다 |
| `ApiV1Filter` 미이식 | **동의(MINOR)** | 게이트웨이 `legacy` 라우트가 `/api/v1` 만 넘기는 것 확인 |

**"이 중 P5 에서 대량 불일치를 만들 것"은 없다. 대량 불일치의 원인은 신고되지 않은 1번(+3번)이고,
P5 가 아니라 P1 에서 터진다.** 골든셋 기준 `/api/v1` 369건 중 최소 92건이 200→401, 60건이
내용만 달라진 200 이다.

## 범위(고아) 점검

변경 파일 전부가 R1(`settings.gradle`·`app/build.gradle`·`modules/legacy/**`·`LegacyModuleMountTest`) /
R2(`RestrictedPathPolicy`·`TokenVerificationFilter`·`LegacyPathAuthPolicy`·`LegacyPathAuthPolicyTest`·
`legacy-auth-boundary.tsv`) / R11(SQL 3종)에 매핑된다. **R# 에 안 붙는 변경은 없다.**
게이트웨이(`Routes.java:34` = `ServiceId.LEGACY` 유지)·`medilawyer-boot` 무수정 확인.
문제는 표에 두 파일이 빠진 것(5번)이지 범위 이탈이 아니다.

## 착수 조건 (P1 전에 닫을 것)

1. **1번**을 닫고, 유효 토큰 1개로 `/api/v1` 토큰필요 경로를 구·신 양쪽에 찔러 200/200 을 실측할 것.
   무토큰 401/401 은 증거로 인정하지 않는다.
2. **3번**을 1번과 같이 처리(1번만 고치면 3번이 드러난다).
3. **7번** 다중행 매핑 애노테이션 전 컨트롤러 스윕 후 대조표·픽스처 재생성.
4. **4번** p1b 섹션 D 트랜잭션 래핑 + `DROP TABLE` 승인자 명시.
5. **5·6번** 문서 인벤토리·AC2 계상 정정.

VERDICT: REQUEST_CHANGES
