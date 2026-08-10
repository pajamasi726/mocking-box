# Round 2 — P0 델타 검토 (수정 권한 `review` — 코드 한 줄도 안 건드렸다)

라운드1 지적 11건이 실제로 닫혔는지, 수정 과정에서 새 결함이 생겼는지만 봤다. 근거는 `git diff HEAD` +
신규 파일 직접 읽기 + 구 `medilawyer-boot@930c83cf`(로컬 체크아웃, 커밋 해시 확인함) 원본 대조 +
개발계 읽기전용 실측이다.

**결론부터: 라운드1 11건은 전부 닫혔다. 내가 재현해 본 것도 다 맞았다.** 대신 작성자가 "미판정 ①"로
남긴 JWT 서명키 항목이 — 확인 불가가 아니라 — **확인 가능하고, 실제로 어긋나 있다.** 그게 이 리뷰의 전부다.

---

## 라운드1 11건 — 닫힘 확인

**1 CRITICAL(토큰 출처)** 닫혔다. `LegacyHeaderTokenExtractor.java:73-82` 가 `owns()` 로 `/api/v1` 만
집어 `Authorization` 을 읽고, 나머지는 손대지 않았다. 주입 순서 문제(아래 질문1)도 실제 조립
컨텍스트에서 단언된다. `LegacyTokenSourceTest:148-154` 의 "쿠키만 실으면 401" 이 특히 좋다 —
"헤더도 읽게" 가 아니라 "구와 같은 곳에서만 읽게" 를 못박는다.

**2·3** 닫혔다(신원 값까지 단언: `:165-173`). **4** `p1b:3637 BEGIN` / `:3658 DROP` / `:3769 COMMIT`
— 재생성이 같은 트랜잭션 안에 들어갔다. **5** 표 16파일이 `git status --short` 실측(수정 3 + 신규 13)과
정확히 일치한다. **6** TSV 130행 = 일치 116 + 의도적 불일치 14(`/internal`), 자기모순 해소.
**8** `TokenVerificationFilter.java:114` `|| owningPolicy != null` + redirect-url 채운 테스트(`:249-258`).
**9** 3케이스 추가 — 특히 `:295-307` 이 등록 순서와 `@Order` 를 일부러 어긋나게 둬서
`orderedStream()`→`stream()` 퇴행을 잡는다. **10·11** 반영.

**7(excel 누락)은 작성자 말을 믿지 않고 구 소스에서 직접 다시 셌다.** 결과가 다 맞는다:
- `@(Get|Post|Put|Delete|Patch|Request)Mapping(` 중 같은 줄에서 괄호가 안 닫히는 것 = **`TeamController.kt:70` 딱 1건**. "전 컨트롤러 스윕 결과 1건뿐"은 사실이다.
- 구 저장소 `src/main` 전체 메서드 레벨 매핑 **131건**, 클래스 레벨 `@RequestMapping` **19건**.
  131 중 1건은 `module-client/.../NotificationService.kt:14` = `@FeignClient` **아웃바운드** 인터페이스라
  엔드포인트가 아니다 → 실엔드포인트 **130**. `P0-auth-boundary.tsv` 130행과 정확히 맞는다.
- 대조의 독립성도 확인했다. `P-1-endpoint-table-map.tsv`(149행 = 130 + "엔드포인트 아님" 19)와
  `P0-auth-boundary.tsv`(130행)를 내가 직접 `comm` 으로 양방향 차집합 냈고 **0/0**. 두 표는 다른 단계에서
  다른 방법으로 만들어졌고(P-1 은 라운드1 이전 산출물이며 라운드1 리뷰 자신이 둘의 불일치를 근거로 썼다)
  순환논증이 아니다.
- `147` 은 `00-log.md:114-115` 에서 이미 "클래스 레벨 접두 19건 중복 계상"으로 정정돼 있다. 130 + 19 = 149 로 닫힌다.

---

## 새 지적

### 1. MAJOR — JWT 서명키는 "확인 불가"가 아니다. 확인했고, **다르다.** P1 착수 차단 사유다

작성자는 `round-1-P0-response.md:91-94` 에서 *"구는 config-server+Vault 에서 받고 `/legacy-service/...`
조회가 `propertySources: []` 로 비어 온다"* → **AC5(P3)로 이월**이라고 적었다. 그 전제가 틀렸다.

구 legacy-service 컨테이너의 엔트리포인트는
`java -Dspring.profiles.active=workstation,git,vault …` 이고, `/app/resources/` 에는
`application{,-dev,-prod,-workstation}.yml` **4개뿐**이다(bootstrap 없음, `spring.config.import` 에
configserver·vault 없음 — `application.yml:12-17` 은 core-client/common/storage 3개만 임포트한다).
즉 유효 출처는 **`application-workstation.yml:18` 의 `medilawyer.jwt.accesstokenSecretKey`** 이고
config-server 는 이 값에 관여하지 않는다. `propertySources: []` 는 대조 불가의 근거가 못 된다.

값 자체는 출력하지 않고 sha256 앞 16자로만 대조했다:

| 대상 | 출처 | sha256(앞16) | 길이 |
|---|---|---|---|
| 구 legacy-service(배포본, 컨테이너 안) | `/app/resources/application-workstation.yml` | `ba5da75eb66bc8cd` | 55 |
| 구 저장소 원본 | `medilawyer-boot@930c83cf .../application-workstation.yml:18` | `ba5da75eb66bc8cd` | 55 |
| 신 booster 저장소 기본값 | `application-dev.yml:158` · `application-localdev.yml:50` | `ba5da75eb66bc8cd` | 55 |
| **신 실행 중 booster-web** | `legalcare-local-booster-web-5` 의 `JWT_SECRET` (= `/mnt/ex_disk1/renew-replay/.env` `SHARED_JWT_KEY`) | **`bd18abfff82b7904`** | **44** |

키 파생은 양쪽이 바이트 단위로 같다 — 구 `JwtProvider.kt:35` `Keys.hmacShaKeyFor(Decoders.BASE64.decode(...))`,
신 `TokenUtils.java:14-17` 동일. **문자열이 다르면 HMAC 키가 다르고, 구 발급 토큰은 신에서 서명 불일치로
401 이 된다**(`TokenVerifyHandler.java:62-64` → `INVALID_TOKEN_SIGNATURE`).

**깨진 것은 코드가 아니라 배포 env 다.** 저장소 기본값은 구와 일치한다. `local` 프로파일에는
`application-local.yml` 에 `jwt` 키가 없어 `application.yml:291` 의 placeholder 로 떨어지고,
compose(`docker-compose.yml:128`)가 `JWT_SECRET: ${SHARED_JWT_KEY}` 로 덮는다. 그 `SHARED_JWT_KEY` 는
신규 스택 4개 서비스가 공유하는 값이라 구 키로 맞추는 것이 다른 서비스에 영향이 없는지도 같이 봐야 한다.

**왜 P3 이월이 안 되나.** AC3 은 *"구 200 → 신 비200 0건"* 이고 P1 대상(`PostController`·`PostReplyController`·
`CommunityController` 등)에는 AUTHENTICATED 경로가 들어 있다. 리플레이는 캡처된 `Authorization` 을
**그대로** 재생한다. 키가 다르면 그 전건이 401 이고, AC3 의 유일한 실패 조건에 정확히 걸린다.
AC5(P3)는 "발급"까지 포함한 4조합이지 검증 절반을 P1 이후로 미루는 근거가 아니다.

라운드1 착수조건 #1이 *"유효 토큰 **1개**로 구·신 양쪽에 찔러라"* 였는데, 재측정은 각 서버가 **자기 키로
서명한 토큰**을 쓴 두 번의 반쪽 측정으로 대체됐다(`round-1-P0-response.md:93`). 그 한 번의 교차가
정확히 이 항목을 잡는 측정이었다.

**조치(코드 수정 불필요)**: ①`SHARED_JWT_KEY`/`JWT_SECRET` 정합 여부를 결정하고 ②구가 발급한 토큰 1개를
booster 에 그대로 찔러 401 이 아님을 실측한 뒤 ③문서의 "미판정 ①"을 실측 결과로 교체. P1 착수조건에 명시.

### 2. MAJOR — `rejectsInvalidToken()` 이 1번의 폭발 반경을 넓혔다 (순서 문제)

`TokenVerificationFilter.java:101` + `LegacyPathAuthPolicy.java:146-148`. 지적3에 대한 **올바른 수정**이고
반박하지 않는다. 다만 결과적으로, 키가 어긋난 상태에서는 permitAll 49경로까지 서명오류로 401 이 된다.
수정 전이었다면 같은 요청이 `:105` 의 warn 후 진행 → 200(익명)이었다. 즉 이번 라운드의 수정이
**"조용한 바디 차이"를 "하드 401"로 승격**시켰다 — 키가 맞을 때만 옳은 동작이다.

골든셋 `/api/v1` 요청 369건 중 `Authorization` 보유 **152건**이 전부 이 분기를 탄다(내 실측, 라운드1 표의
92+60과 정확히 일치). 1번을 P1 전에 닫으면 그대로 두는 것이 맞고, 안 닫으면 P1 이 대량 회귀로 오독된다.
**둘의 선후를 문서에 못박을 것.**

### 3. MINOR — `LegacyHeaderTokenExtractor.java:25-26` 의 골든셋 수치가 요청 수가 아니라 문자열 출현 수다

javadoc이 *"`/api/v1` 레코드 4,186건 중 authorization 3,963 / cookie 0"* 을 실측으로 적어 뒀다.
35파일을 `type=entry` 기준으로 파싱해 보면 총 엔트리 118,233건, 그중 **요청 경로가 `/api/v1` 인 것은 369건**,
`authorization` 152 / `cookie` **0** 이다. 4,186 은 `/api/v1` **문자열이 들어간 줄** 수고(응답 바디·Referer 등
포함), 3,963 도 같은 방식이다.

**결론(cookie 0 = 헤더 전용)은 안 바뀐다.** 문제는 이 값이 운영 소스 javadoc에 "실측"으로 박혔고
같은 태스크의 라운드1 표(92+60=152)와 11배 어긋난 채 병존한다는 것이다. `round-1-P0-response.md:5` 도 같다.
숫자만 정정하면 된다.

### 4. MINOR — `/api/v2` 는 "두 판정이 이미 같은 답"이 아니다. 토큰 출처가 갈린다

`LegacyPathAuthPolicy.java:49-54` 는 **restricted/permitAll 판정**이 같다는 것만 논증하고 "건드릴 이유가
없다"로 닫는다. 그런데 구 `JwtAuthenticationFilter` 는 `OncePerRequestFilter` 로 **경로 제한 없이**
전 요청에서 `getHeader("Authorization")` 을 읽는다(`:29-39`) — `/api/v2` 도 헤더였다. 신은
`CookieTokenExtractor.java:23` 이 잡아 쿠키다. **라운드1 CRITICAL 과 똑같은 모양의 반쪽 분석**이
`/api/v2` 에 남아 있고, `LegacyTokenSourceTest:231-242` 가 그 쿠키 동작을 "의도"로 못박아 버렸다.

다만 골든셋 35파일에 `/api/v2` 요청은 **0건**(내 실측)이라 리플레이로는 영영 안 잡히고, `/api/v2` 는
P0 범위 밖(선행 태스크 산출물)이다. 그래서 MINOR 다. 코드가 아니라 기록의 문제 — 지금 문구는
"확인했고 문제없다"로 읽힌다. **소유 태스크로 넘기는 이월 항목으로 남길 것.**

### 5. MINOR(NIT) — `findOwningPolicy()` 가 요청마다 `orderedStream().toList()` 를 돈다

`TokenVerificationFilter.java:161`. 10개 모듈 전 요청이 타는 자리에서 매 요청 빈 조회 + LinkedHashMap +
정렬 + 리스트 할당이다. 정책 빈은 싱글턴이고 개수가 런타임에 안 변하니 한 번만 물려도 된다. 기능 문제 없음.

---

## 오케스트레이터 질문에 대한 판정

**질문1 — `@Order` 우선순위가 깨질 수 있는가.** 깨지지 않는다. `List<T>` 의존성 주입은
`DefaultListableBeanFactory` 가 `dependencyComparator`(= `AnnotationAwareOrderComparator`,
`AnnotationConfigUtils` 가 모든 애노테이션 컨텍스트에 등록)로 **정렬해서** 넘긴다. `@Order` 없는 기존 두
추출기는 `LOWEST_PRECEDENCE` 라 `HIGHEST_PRECEDENCE` 가 항상 앞선다 — 등록 순서와 무관하고 결정적이다.
프록시가 껴도 안전하다(`OrderUtils` 가 `MergedAnnotations` TYPE_HIERARCHY 로 읽고,
`FactoryAwareOrderSourceProvider` 가 타깃 클래스를 별도 order source 로 준다). 조건부 빈으로 추출기가
아예 안 뜨는 경우가 유일한 실패 모양인데, `LegacyModuleMountTest:102-125` 가 **실 조립 컨텍스트에서
주입된 필드를 리플렉션으로 읽어** 0번이 `LegacyHeaderTokenExtractor` 임을 단언하므로 CI 에서 걸린다.
`LegacyTokenSourceTest:76-79` 는 리스트를 손으로 만들어 순서 회귀를 못 잡는데, 작성자가 `:74-75` 주석에
그렇게 적어 두고 진짜 가드를 다른 테스트로 돌렸다 — 정직하고 맞는 배치다. 다만 그 가드가 **한 개**다.

**기존 9모듈 추출 경로**는 한 건도 안 바뀐다. `owns()` 는 `/api/v1` 접두만 주장하고(`LegacyPathAuthPolicy:116-124`),
booster 전 모듈에서 `/api/v1` 을 **서빙**하는 매핑은 0건이다(내가 다시 훑었다 — `modules/crawling/.../LegacyService.java:16`
은 구를 호출하는 아웃바운드 인터페이스, `modules/user/.../StoreController.java:53` 등은 Swagger `description`
문자열이다).

**질문2 — 404 가 "게이트 통과"의 증명이 되는가.** **성립한다. 단 범위가 좁다.** 필터 흐름상
AUTHENTICATED 경로에서 토큰이 공백이면 `:107-115` 로 401, 검증 실패면 `:101-102` 가 던져 401 이다.
404 는 `chain.doFilter` 까지 갔다는 뜻이므로 **토큰이 비어 있지 않았고 `tokenVerify` 가 예외 없이
끝났다**는 것까지 증명한다(비-`LegalCareException` 이 나면 500 이지 404 가 아니다). 401→404 델타가
헤더를 읽기 시작했다는 증거로 유효하다.
증명하지 **못하는** 것: ①신원 속성 값이 옳은지 — 이건 `LegacyTokenSourceTest:140-143` 의 바디 단언이
따로 덮는다 ②**그 토큰이 구가 발급한 토큰인지** — 측정에 쓰인 것은 booster 자기 키로 서명한 토큰이므로
지적 1이 남는다. 즉 404 는 "booster 가 자기 토큰을 헤더에서 읽어 검증한다"까지고, "구 토큰이 신에서
검증된다"는 여전히 미증명이다.

**질문3 — 미판정 2건의 심각도.** 서명키 = **MAJOR, P3 아니라 P1 착수 차단**(지적 1·2).
만료 토큰 구 400 vs 신 401 = **MINOR, P6 전 결정 동의.** 공용 catch(`:128-131`)가 9모듈 전체의 인증 실패
응답을 401 로 고정하고 있어 여기서 바꾸면 범위를 벗어난다. 기능적 차이(거부/서빙)는 닫혔고 방향도
`구 400 → 신 401` 이라 AC3 의 단방향 규칙에 안 걸린다. 401 바디 차이와 묶어 올린 판단이 맞다.

**질문4 — `rejectsInvalidToken()` 이 기존 9모듈에 영향 가는가.** 안 간다. 조건이
`owningPolicy != null && …` 이고 `owns()` 가 `/api/v1` 뿐이라 다른 경로는 `owningPolicy == null` →
표현식이 short-circuit 으로 false → 종전 경로 그대로다. 기본값도 `false`(`RestrictedPathPolicy:63-65`).
`LegacyTokenSourceTest:206-214` 가 "다른 모듈 unrestricted 경로는 만료 토큰을 삼키고 진행"으로 못박았다.
(지적 2는 무회귀가 아니라 `/api/v1` **자체**의 위험이다.)

**질문5 — 130↔130 대조가 독립적인가.** 독립적이다. 위 "7" 참조 — 작성자 산출물 둘을 서로 대조한 것에
더해, **구 소스에서 내가 직접 센 131개 매핑 애노테이션 − Feign 1 = 130** 으로 제3의 기준과도 맞았다.

**질문6 — 범위 이탈.** 없다. 수정 패스가 더한 것은 `LegacyHeaderTokenExtractor.java`(지적1) ·
`LegacyTokenSourceTest.java`(지적1) · `app/build.gradle` 의 `jjwt-api` **테스트** 의존(지적1 측정용) ·
p1b 트랜잭션 래핑(지적4) 뿐이고 전부 지적에 대응한다. 16파일 전부 R1/R2/R11 에 매핑되고 고아 없음.
게이트웨이 `Routes.java:34` = `ServiceId.LEGACY` 유지, `medilawyer-boot` 무수정 확인.

---

## 불변조건 (내가 한 것)

개발계는 읽기만 했다 — `docker ps`/`docker inspect`, 컨테이너 안 `cat`/`grep` 1회,
`/mnt/ex_disk1/prod-capture` 의 골든셋 파싱(읽기). 컨테이너 정지·삭제·compose 조작 0, 운영 RDS 접근 0,
구·신 어느 서비스에도 HTTP 요청 0(외부 실발사 없음), 코드 수정 0, 커밋 0. 키 값은 어디에도 출력하지 않고
sha256 앞 16자만 썼다.

## P1 착수 전에 닫을 것

1. **지적 1** — 구 발급 토큰 1개를 booster 에 그대로 찔러 서명 검증이 통과함을 실측. 안 통과하면
   `SHARED_JWT_KEY` 정합(다른 3개 서비스 영향 확인 포함). 문서의 "미판정 ①" 교체.
2. **지적 2** — 1번과의 선후를 P1 착수조건에 명시.
3. **지적 3** — javadoc·응답서의 골든셋 수치를 요청 기준(369 / 152 / 0)으로 정정.
4. **지적 4** — `/api/v2` 토큰 출처 divergence 를 소유 태스크 이월 항목으로 기록.

코드 수정이 필요한 항목은 하나도 없다. 4건 다 실측 1회 + 문서 정정이다.

VERDICT: REQUEST_CHANGES
