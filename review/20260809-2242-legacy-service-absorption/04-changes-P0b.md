# 변경 요약: P0b 게이트웨이 인증 집중 (R2 인증경계 이설의 연장 · 사용자 12:45 결정)

브랜치 `feature/prod-sync-candidates` @ `/Users/steve/steve/legalcare-renew-prodsync-wt` · **커밋 0건**

## 착수 전제 2건이 실측과 다르다 — 먼저 정정한다

**① "게이트웨이 인증이 꺼져 있다"는 사실이 아니다.** 게이트웨이가 읽는 프로퍼티는 `JWT_SECRET` 이
아니라 `JWT_TOKEN_KEY` 다(`gateway-app/application.yml:27` = `${JWT_TOKEN_KEY:}`). 실행 중 컨테이너에는
그 값이 **있다**:

```
$ ssh -p 50022 legalcare@114.203.1.178 'docker inspect -f "{{range .Config.Env}}{{println .}}{{end}}" legalcare-local-gateway-1'
JWT_TOKEN_KEY=f9raBLU/NcGISlDZBdRpzRxd+F1Zvn6BlxhCeS8Vp4E=      <- sha256 앞16 = bd18abff, 44자
SPRING_PROFILES_ACTIVE=dev
$ docker logs legalcare-local-gateway-1 | grep "JWT_TOKEN_KEY 미설정"     -> (빈 출력)
```
키가 있고 기동 경고도 없다 → `JwtVerifier.enabled()` = **true**. 즉 게이트웨이는 지금도 검증하고
`X-Auth-*` 를 주입하고 있으며, 그 키는 실행 중 3앱의 `JWT_SECRET`(`bd18abff…`/44)과 **같은 값**이다.

**② 그런데 `/api/v1` 에서는 한 번도 주입된 적이 없다 — 원인은 키가 아니라 토큰 출처다.**
`IdentityHeaderFilter` 가 `ApiPathMatcher.isApiPath()` 로 헤더/쿠키를 가르는데 그 매처는 `/apis`·
`/{module}/apis` 만 참이라 `isApiPath("/api/v1/...")` = **false** → `/api/v1` 은 **쿠키**를 찾는다.
구 클라이언트는 쿠키를 안 보낸다(골든셋 `/api/v1` 4,186건 중 authorization 3,963 / cookie 0).
**P0 라운드1의 CRITICAL 과 같은 결함이 게이트웨이에 그대로 있었다.** 실행 중 이미지 소스로 확인:
```
$ git show 8b1f7ec:src/main/resources/application-dev.yml | grep jwt      -> (없음)
$ git show 8b1f7ec:.../auth/IdentityHeaderFilter.java | sed -n '44p'
        if (ApiPathMatcher.isApiPath(request.getRequestURI())) {
```
→ 이 단계의 실제 과제는 "키 주입"이 아니라 **토큰 출처 정정 + 다운스트림 수신부 신설**이었다.

## 변경 파일
| 파일 | 한 줄 요약 | R# |
|---|---|---|
| `gateway-app/.../auth/IdentityHeaderFilter.java` | `/api/v1` 도 Authorization 헤더에서 읽는다 + 신뢰 표식 동봉 | R2(P0b) |
| `gateway-app/.../auth/IdentityHeaders.java` | `X-Auth-Trust` 추가(신원이 있을 때만) | R2(P0b) |
| `gateway-app/.../config/GatewayProperties.java`·`GatewayConfig.java` | `gateway.identity.trust-secret` 배선 | R2(P0b) |
| `gateway-app/src/main/resources/application.yml` | 위 키 선언(빈 기본값 = 꺼짐) | R2(P0b) |
| `gateway-app/src/main/resources/application-dev.yml` | `gateway.jwt.key` dev 기본값을 앱 3종과 동일하게 | R2(P0b) |
| `booster-app/core/.../filter/TrustedIdentityResolver.java` | 신규 확장점(빈 0개면 종전 경로) | R2(P0b) |
| `booster-app/core/.../filter/TokenVerificationFilter.java` | 신뢰 신원 분기 + 애트리뷰트 승격 블록 추출 | R2(P0b) |
| `booster-app/app/.../security/LegacyGatewayIdentityResolver.java` | `X-Auth-*` → `appUserId`/`email`/`organizationRole` (시크릿 검증 후에만) | R2(P0b) |
| `booster-app/app/src/main/resources/config/legacy.yml` | `legacy.identity.trust-secret`(빈 기본값 = 꺼짐) | R2(P0b) |
| `gateway-app` 테스트 3파일 · `booster-app` 테스트 3파일 | 아래 "테스트 결과" 참조 | R2(P0b) |

P0 의 `LegacyHeaderTokenExtractor`·`LegacyPathAuthPolicy` **무삭제**(과도기 폴백으로 유지).
`Routes.java:34` 무변경(`ServiceId.LEGACY`) · `medilawyer-boot` `git status` 빈 출력 · R# 미매핑 변경 없음.

## 테스트 결과
```
booster  기준선 ./gradlew test --rerun-tasks  -> tests=729 failures=0 errors=0 skipped=0
         변경 후                              -> tests=744 failures=0 errors=0 skipped=0   (+15, 전부 app)
         app 223 core 48 chrono 7 crawling 68 external 40 legacy 29 medicontents 18
         notification 87 post 67 product 71 user 86
         ./gradlew duplicateClassCheck        -> DUP_EXIT=0
gateway  기준선 76/0  ->  변경 후 84/0 (+8)
```
**뮤테이션 5건 전부 문다**(각 1회 되돌린 뒤 재실행, 전부 원복 확인):
| 뮤테이션 | 결과 |
|---|---|
| GW `/api/v1` 헤더 추출 제거 | 84/**4 fail** — IT 2건 + 필터 2건 |
| GW 인바운드 `X-Auth-*` 스트립 제거 | 84/**5 fail** — IT `forged_identity_without_token_never_reaches_downstream` 포함 |
| booster `trusted()` 항상 true | **2 fail** — 신뢰 헤더 없음/불일치 위조가 통과 |
| booster 기능 스위치(빈 시크릿 조기 반환) 제거 | **1 fail** — `X-Auth-Trust: ""` 하나로 뚫림(`MessageDigest.isEqual([],[])==true`) |
| core 신뢰 신원 분기 제거 | **2 fail** — 신뢰 헤더가 무시됨 |

## 과제별 자가 점검
**1. 게이트웨이 인증 활성화 경로** ✅ 저장소 설정으로 확정. `gateway.jwt.key` ← `JWT_TOKEN_KEY` 이고
dev 프로파일에만 기본값을 넣었다(앱 3종 `application-dev.yml` 과 같은 리터럴 = 구 access 키 `ba5da75e…`/55).
release 에는 넣지 않았다 — 운영이 env 를 잊었을 때 공개 리포 키로 검증하면 안 된다. 컨테이너 무접촉.

**2. 다운스트림 수신부** ✅ **없었다 → 만들었다.** 전수 grep 결과 `X-Auth` 는 `gateway-app` 안에만 존재했고
booster/reviewed/lawkit 에 소비자가 0건이었다(= 게이트웨이가 넣어 준 헤더를 아무도 안 읽고 있었다).
`LegacyGatewayIdentityResolver` → core 훅 → `TokenVerificationFilter` 가 종전과 **같은 애트리뷰트**를 심으므로
`LegacyUserDetailsArgumentResolver`·`LegacyAppUserResolutionInterceptor` 는 한 줄도 안 바뀐다.
우선순위는 지시대로 **신뢰 헤더 우선 → 없으면 자체 검증 폴백**(`noTrustHeader_fallsBackToSelfVerification`).

**3. refresh 토큰 경로** ✅ 설계만. **구현하지 않은 근거는 아래 별도 절.**

**4. 스푸핑 방어 실증** ✅ 실행으로 증명. 게이트웨이 경유는 실 Tomcat E2E(`GatewayApplicationIT`)로,
게이트웨이 우회는 booster 필터 체인 직접 투입(`LegacyGatewayIdentityTest`)으로 갈랐다. 코드 읽기 갈음 없음.

**5. 다운스트림 자체 검증** ✅ **남긴다**로 판정. 근거는 아래 별도 절.

## refresh: 게이트웨이가 담당할 수 없다 (설계 결론, 구현은 P3/R5)
구 `POST /api/v1/jwt/refresh`(`JwtController.kt:41-46` → `JwtProvider.kt:81-98`)가 하는 일 4가지 중
**순수 암호 연산은 1개뿐**이다: ①refresh 키로 서명 검증 ②`token` 테이블을 email 로 조회해 저장된
refresh 값과 **문자열 동일성 비교**(`TokenRepository.findByRefreshTokenKeyAndUsable`) ③`organization_app_user`
재조회 ④access+refresh 재발급 + `token` upsert. ②③④가 전부 DB 상태다. 게이트웨이에는 데이터소스가
없다(`gateway-app/build.gradle:14-28` — web·actuator·jjwt 뿐). 게다가 **구는 refresh 토큰을 헤더가 아니라
바디로 받는다**(`RefreshReqDto`) — 무변형 스트리밍 프록시는 바디를 파싱하지 않으므로 검증 자체가 불가능하다.
→ **refresh 전용 키를 게이트웨이에 둘 이유가 없다. 앱에 둔다.**

키 배치 결론: access 키 = 게이트웨이(검증) + 앱(발급) 양쪽 같은 값 / refresh 키 = **앱만**.
P3 착수 시 반드시 해야 할 일: 신규는 `jwt.key` 하나뿐이므로 구 `medilawyer.jwt.refreshtokenSecretKey`
(`a1fa3d59…`/55)에 대응하는 프로퍼티를 **새로 열어야** 한다. 안 열면 구가 발급한 refresh(만료 30일,
`JwtProvider.kt:143`)를 신이 검증하지 못해 **흡수 후 기존 세션이 최대 30일 안에 전부 강제 로그아웃**된다.
이번에 구현하지 않은 이유는 지시서가 `JwtController`·`AuthController`·`OtpController` 를 **R5(P3)** 로
배정해 뒀기 때문이다 — 여기서 손대면 R# 미매핑 변경이 된다.

## 판정: 다운스트림 자체 검증을 남긴다
사용자 목표는 "키는 게이트웨이에만"인데 **그건 발급까지 옮기지 않는 한 성립하지 않는다.**
`createToken` 호출부가 로그인(`AuthService.kt:76,93`)·초대수락(`OrganizationAppUserFacade.kt:88`)·
`JwtController.kt:35` 로 전부 앱에 있고, 위 refresh 이유로 그것들은 앱에 남는다. **발급하는 쪽은 서명키를
계속 들어야 한다.** 키가 앱에 있는 이상 게이트웨이 단독 검증은 보안 이득이 없고 신뢰 경계만 하나 늘린다.

그래서 이렇게 했다: **헤더 소비 경로는 만들되, 공유 시크릿 뒤에 두고 기본값을 꺼짐으로 둔다.**
`legacy.identity.trust-secret` 이 비면 리졸버가 헤더를 아예 보지 않으므로 **오늘의 동작이 한 줄도 안 바뀐다**
(실 조립 컨텍스트에서 단언 — `LegacyModuleMountTest#legacyGatewayIdentityResolver_isRegisteredButInertByDefault`).
시크릿을 안 쓰고 헤더를 그냥 믿으면 개발계 `127.0.0.1:18081`·컨테이너망 `http://booster-web:18080` 에
붙을 수 있는 누구나 임의 사용자로 행세한다 — 게이트웨이의 인바운드 스트립은 **게이트웨이를 지나올 때만**
유효하고, 이 저장소는 서버간 직결을 의도적으로 열어 뒀다(`ProxyFilter` javadoc). 그래서 시크릿이 필수다.

## 적용 절차 (개발계 컨테이너는 이번에 건드리지 않았다)
1. **게이트웨이 검증 키를 구 access 키로 맞춘다** — 지금은 게이트웨이·신규3앱이 `bd18abff…`(44)로 일치하고
   구 legacy-service 만 `ba5da75e…`(55)다. 게이트웨이가 **구가 발급한 토큰**도 검증하려면 `JWT_TOKEN_KEY` 를
   구 access 키로 바꿔 재생성해야 한다. ⚠️ `deploy/dev/docker-compose.yml:39` 이 `${DEV_JWT_KEY}` 를
   **명시로 넘기므로**, 그 값이 비면 env 가 "빈 문자열로 존재"해 `${JWT_TOKEN_KEY:기본값}` 의 기본값이 **안 먹는다**.
   → `.env` 의 `DEV_JWT_KEY` 를 채우거나 그 줄을 지워야 저장소 기본값(이번에 넣은 dev 기본값)이 산다.
2. **신뢰 헤더를 켠다(선택)** — `GATEWAY_IDENTITY_TRUST_SECRET`(GW) = `LEGACY_IDENTITY_TRUST_SECRET`(booster),
   같은 ASCII 값. **한쪽만 켜도 안전하다**(한쪽이 비면 신뢰 성립 X → 자체 검증 폴백) → 순서 제약 없음.
3. 게이트웨이 이미지 재빌드가 선행돼야 한다 — 실행 중 이미지에는 `/api/v1` 헤더 추출 수정이 없다.

## 알려진 한계 / 리뷰어에게
1. **실 게이트웨이 ↔ 실 booster 로 신뢰 경로를 통과시켜 보지 못했다.** 컨테이너 재생성 금지 때문이다.
   양쪽을 각각 실행 검증했고 헤더 이름 계약은 리터럴로 못박았지만(`headerNames_matchGatewayContract`),
   **두 프로세스가 실제로 맞물리는 것은 미검증**이다. 두 앱은 서로 의존하지 않아 컴파일러가 이 계약을
   지켜 주지 않는다 — 이름이 어긋나면 에러 없이 "신원이 조용히 안 넘어온다". P5 진입 조건에 넣을 것.
2. **`X-Auth-Trust` 는 정적 공유 시크릿이다.** 사설망 트래픽을 관찰할 수 있는 자가 시크릿을 얻으면
   임의 신원을 만들 수 있다. 값에 대한 HMAC 으로 올리면 관찰자는 본 값 그대로만 재생할 수 있게 되지만,
   이번엔 넣지 않았다 — 오늘 `/internal/**` 이 이미 무인증으로 사설망에 열려 있어(`bypass.url` 에 `internal`)
   같은 등급의 노출이 이미 존재하고, 단계 범위를 넘는 설계 결정이라 판단했다. **결정 항목으로 올린다.**
3. **`reviewed`/`lawkit` 은 이번 범위 밖이다.** 세 앱의 `core` 는 패키지가 다른 **별개 사본**이라
   (`legalcare.medilawyer.core` / `legalcare.reviewed.core` / `legalcare.lawkit.core`) 이번 core 변경의
   영향은 booster 10모듈에 국한된다. 두 앱에도 같은 구조가 필요해지면 각자 사본에 같은 훅을 넣어야 한다.
4. **`gateway.jwt.key` 기본값을 dev 프로파일에 넣은 것은 양날이다.** 검증 키의 공개 기본값이라 서명 키만큼
   위험하진 않지만, 신뢰 헤더가 켜진 상태에서 운영이 `release` 가 아닌 `dev` 로 뜨면 공개 키로 서명한
   토큰이 신원으로 승격된다. `docs/audit/F-config-wiring.md:50,54` 의 🟠 항목과 같은 성격이라 거기 묶어
   관리해야 한다(이번에 그 문서는 고치지 않았다 — 다른 태스크가 공유하는 원본이다).
5. **미판정**: 구 legacy-service 가 실제로 발급한 토큰을 이 게이트웨이가 검증하는지 실물로 재 보지 못했다.
   키가 같으면 성립한다는 것은 코드로 확실하나(같은 jjwt HS + base64 디코딩), 실증은 P3 의 AC5 4조합 몫이다.

STATUS: DONE
