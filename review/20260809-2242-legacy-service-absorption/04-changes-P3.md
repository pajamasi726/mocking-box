# 변경 요약: P3 인증 발급 이식 (R5 — `AuthController`·`OtpController`·`JwtController` 5경로)

브랜치 `feature/prod-sync-candidates` @ `/Users/steve/steve/legalcare-renew-prodsync-wt`

## ⚠ 먼저 — 내가 하지 않은 커밋·푸시가 작업 중에 일어났다

작업 중이던 **19:34**, 외부에서 `git commit` + `git push` 가 실행돼 P3 작업물이
`e9a6cb0 [FEAT] 레거시 서비스 흡수 P-1~P2 — …` 라벨의 커밋에 **통째로 섞여 origin 에 올라갔다**
(387파일). **나는 커밋·푸시를 한 적이 없다.**

```
$ git show -s --format="%h %an %ad %s" e9a6cb0
e9a6cb0 steve Mon Aug 10 19:34:28 2026 +0900 [FEAT] 레거시 서비스 흡수 P-1~P2 — 스키마 동기화·인증 경계·읽기/쓰기 62경로
$ git show --diff-filter=A --name-only --format="" e9a6cb0 | grep -cE "legacy/(auth|db/domain/otp|db/dto/auth|…)"
23                       # P3 신규 main 파일이 P-1~P2 커밋에 들어가 있다
$ git ls-remote --heads origin feature/prod-sync-candidates
e9a6cb0…  refs/heads/feature/prod-sync-candidates          # 푸시됨
$ git reflog show origin/feature/prod-sync-candidates | head -1
e9a6cb0 refs/remotes/origin/…@{0}: update by push
```

결과 3가지: ① **R10 "단계마다 커밋 1개" 가 깨졌다**(P3 가 P-1~P2 커밋 안에 있다) ② 커밋 메시지가
실제 내용과 다르다 ③ 그 시점의 P3 는 **전체 테스트도 리뷰도 끝나기 전**이었다. 히스토리 정리는 내
권한 밖이라 손대지 않았다 — **결정 요청 항목**. (부수 효과 하나는 좋다: 뮤테이션 14건을 돌린 뒤
`git status` 가 그 파일들을 **무변경**으로 보고하므로 원복이 바이트 단위로 확인된다.)

**푸시된 트리 ≠ 지금 워크트리**다. 커밋 이후에 만든 것 2건이 아직 미커밋이다 —
`AuthControllerTest.java`(신규) · `LegacyJwtCrossVerificationTest.java`(위 전제 정정 반영분).
리뷰는 워크트리 기준으로 봐야 한다(`git status --short` 2줄).

## 착수 전제 정정 — **구의 access·refresh 키는 개발계에서 사실상 같은 키다**

지시서·`00-log.md 12:30` 은 "구는 access·refresh 를 서로 다른 키로 서명한다"를 전제로 P3 를 짰다.
**문자열은 다르다. 그런데 jjwt 가 실제로 쓰는 것은 `Decoders.BASE64.decode` 한 바이트다.**
두 문자열은 **마지막 1글자만** 다르고, base64 는 마지막 문자의 남는 비트를 버린다:

| 프로파일 | raw sha256[:16] access/refresh | raw 가 다른 위치 | **decoded** sha256[:16] access/refresh | 같은 키? |
|---|---|---|---|---|
| workstation(구 개발계 실동작) | `ba5da75e…` / `a1fa3d59…` (55/55) | **인덱스 54 하나** | `7d0bc947…` / `7d0bc947…` (41/41) | **예** |
| dev | 동일 | 동일 | 동일 | **예** |
| prod | `ecce27e0…` / `dc57acbb…` (75/76) | 8곳 | `a8109666…` / `e6af6179…` (56/57) | **아니오** |

→ **개발계에서는 refresh 축이 존재하지 않는다**(그래서 신규의 단일 `jwt.key` 하나로도 구 refresh 가
열린다 — "흡수 후 30일 강제 로그아웃" 은 개발계에선 안 일어난다). **운영에서는 진짜로 두 축이다.**
그래서 P3 는 두 프로퍼티를 따로 열었고, 이 사실을 가정이 아니라 **런타임에 재서** 단언한다
(`LegacyJwtCrossVerificationTest` 의 `[2-진단]`). 개발계 값 하나만 회전해도 축이 갈라지므로,
"오늘 같으니 하나로 합치자"는 반대 방향이다.

## 변경 파일

| 파일 | 한 줄 요약 | R# |
|---|---|---|
| `L/auth/api/controller/{Auth,Otp}Controller.java` | 3경로 이식(sign-up 201 / login·otp 200) | R5 |
| `L/auth/api/service/{Auth,Otp}Service.java` | 회원가입·로그인·OTP 발송 본체 | R5 |
| `L/security/JwtController.java` | `/jwt/create`(무인증·하드코딩 계정) · `/jwt/refresh`(201) | R5 |
| `L/security/JwtProvider.java` | `reissueJwtSvc`·`getRefreshClaims`·`associateOrganizationIdAndRole` 추가, `OrganizationAppUserService` 의존 | R5 |
| `L/db/domain/otp/{Otp,repository/*}.java` (4) | `otp` 테이블 + upsert + 5분창 조회 2개 | R5 |
| `L/db/dto/auth/*.java` (4) · `L/db/dto/jwt/RefreshReqDto.java` | 요청·응답 DTO(+`@LegacyNonNull` 9필드) | R5 |
| `L/db/dto/bizmessage/Sms{Req,Res}Dto.java` | NCP SENS SMS 요청·응답(+`@LegacyNonNull` 4) | R5 |
| `L/db/dto/kafka/ProducerUserSignUp.java` | 가입 이벤트 페이로드(+마스킹 `toString`) | R5 |
| `L/enums/{OtpType,Sms,SmsType}.java` | 상수표 3종 | R5 |
| `L/client/bizmessage/…/BizMessageService.java` | `sendSmsSvc` + 헬퍼 3 + `smsServiceId` | R5 |
| `L/client/email/…/PushEmailService.java` | `sendOtpEmail` + `awsEamilServerHost` | R5 |
| `L/client/kafka/config/LegacyKafkaProducerConfig.java` | `ProducerUserSignUp` 구 FQCN 타입매핑 | R5 |
| `L/apis/http/appUser/service/AppUserService.java` | `getAppUser(email,status,usable)` (구 5메서드 완성) | R5 |
| `A/main/resources/config/legacy.yml` | `naverCloud.smsServiceId`·`aws.awsEamilServerHost`(**둘 다 스텁/자리표시자**) | R5 |
| 테스트 8파일 | 아래 "테스트 결과" | R5 |

**R# 미매핑 변경 0.** `:core`·게이트웨이·SQL·`medilawyer-boot` 무수정(`git -C medilawyer-boot status` 0줄) ·
`Routes.java:34` = `ServiceId.LEGACY` 불변.

## 이식 대조 (기계 · 빠짐0·유령0)

```
$ python3 scratchpad/p3_parity.py
(1) 엔드포인트: AuthController 2 / OtpController 1 / JwtController 2  — 전부 OK (빠짐0·유령0, 핸들러명까지)
(2) DTO 필드명·선언순서 + @JsonPropertyOrder: 10클래스(중첩 포함) 전부 OK, 불일치 총계 0
$ python3 scratchpad/p3_nonnull.py
구 Kotlin '기본값 없는 non-null' 전수 = 11클래스/23필드
그중 역직렬화 경로(요청바디 4 + 외부응답 1) = 5클래스/13필드에 @LegacyNonNull 부착, 불일치 0
직렬화 전용 6클래스/10필드는 미부착(규칙대로) — 초과 부착 0
```
진입점은 추정이 아니라 실측이다: `grep @RequestBody` → 4종, `grep .body(*.class)` → `SmsResDto` 1종.
접촉 테이블 5개(`otp`·`app_user`·`token`·`organization_app_user`·`organization`)는 `P-1-endpoint-table-map.tsv`
의 auth/otp/jwt 5행과 일치하고 **신규 PG 에 전부 존재**하며 `otp` 에 `unique_email`(→ `ON CONFLICT (email)` 성립)이
있다(psql 실측). 그 표의 `team_app_user/team/store` 는 구 엔티티 LAZY 연관을 따라간 정적 결과이고 이 경로에서
실제로 조회되지 않는다.

## 테스트 결과

```
booster  $ rm -rf */build/test-results modules/*/build/test-results
         $ ./gradlew test duplicateClassCheck --rerun-tasks
           BUILD SUCCESSFUL in 4m 26s   (61 actionable tasks: 61 executed)
           tests=850 failures=0 errors=0 skipped=8            기준선 797 → +53
           app 251 / core 48 / chrono 7 / crawling 68 / external 40 / legacy 107 /
           medicontents 18 / notification 87 / post 67 / product 71 / user 86  (클래스 221)
gateway  ./gradlew test --rerun-tasks   -> tests=84 failures=0 errors=0   (기준선 84 불변)
```
skipped 8 = `LegacyJwtCrossVerificationTest`(실키 env 없으면 클래스 전체 비활성). env 를 주면 8/8 통과한다(아래 AC5).

**뮤테이션 14/14 문다**(각 1건 적용 → 대상 테스트 실행 → 원복, 원복은 `git status` 로 확인):

| 뮤테이션 | 깨지는 테스트 |
|---|---|
| refresh 를 access 키로 서명 / 재발급의 `token` 저장값 비교 무력화 / `associate` 를 "먼저 승리"로 | `JwtProviderRefreshTest` |
| `RefreshReqDto`·`SmsResDto.statusCode` 의 `@LegacyNonNull` 제거 | `LegacyP3ContractTest` |
| SIGNUP 의 `requireNonNull` 제거 / `maskedPhone` 자리수 변경 | `OtpServiceTest` |
| `loginSvc` 판정 순서 뒤집기(OTP 먼저) | `AuthServiceTest` |
| `/jwt/refresh` 201→200 / `jwt/create` 하드코딩 계정 변경 | `JwtControllerTest` |
| `sign-up` 201→200 | `AuthControllerTest` |
| 카프카 `ProducerUserSignUp` 매핑 제거 / 마스킹 `toString` 제거 | `LegacyKafkaProducerConfigTest` · `KafkaDtoLoggingMaskTest` |
| `otp/send` 매핑 경로 변경 | `LegacyP1EndpointMountTest` |

## AC 자가 점검

**AC5 ✅ — 4조합 전부 실측.** 구 41010 에 SSH 터널을 열고 실키를 env 로만 주입해
`LegacyJwtCrossVerificationTest` 8/8 통과. 원문 출력:
```
[AC5] accessSecret sha256[:16]=ba5da75eb66bc8cd len=55 / refreshSecret a1fa3d599cc87fd0 len=55 / 서로 다름=true
[AC5][2-진단] decoded accessKey 7d0bc947910fd46c(41) / refreshKey 7d0bc947910fd46c(41) / 같은 HMAC 키=true
[AC5][2]        구 발급 refresh -> 신 getRefreshClaims = OK, claimKeys=[appUserId,email,tokenName,organizationIdAndRole,exp], exp=2026-09-09
[AC5][3-access] 구 GET /api/v1/app-users/me            status=200 OK
[AC5][3-refresh]구 POST /api/v1/jwt/refresh            errorMessage=유효하지 않은 토큰입니다.   (서명 통과, token 저장값에서 갈림)
[AC5][4]        구 POST /api/v1/jwt/refresh(현재 키 밖) errorMessage=토큰 형식 오류              (라이브 음성 대조군)
```
[3]과 [4]가 **같은 엔드포인트에서 다른 문구로 갈린다**는 것이 판정의 핵심이다 — 구는 서명 실패와
저장값 불일치를 똑같이 HTTP 400 으로 내므로 상태코드로는 못 가른다. [1]신→신은 로컬 왕복,
[2]구→신은 구가 어제(2026-08-09) 발급해 `token` 테이블에 남긴 실물 토큰으로 확인했다.

**OTP·SMS 실발송 0건 ✅.** 구에서 토큰을 새로 발급받는 경로를 **쓰지 않았다** —
`/jwt/create` 는 하드코딩 계정이 개발계에서 `INACTIVE` 라 못 쓰고, login/sign-up 은 OTP 실발송을
태운다. 그래서 이미 저장돼 있던 토큰을 구 발급물로 썼다. **구 DB 쓰기 0건 실측**:
```
프로브 전/후  token 393행 → 393행 · max(updated_at) 2026-08-09 불변 · jay 행 updated_at 2025-11-05 불변 · otp 461행 불변
legacy-service RestartCount=0 running (컨테이너 조작 0 · env 변경 0)
```

## 알려진 한계 / 리뷰어에게

1. **`POST /api/v1/jwt/create` 는 무인증으로 하드코딩 계정(`jay@legalcare.ai`)의 access·refresh 를
   누구에게나 내준다.** 구 그대로 옮겼다(구 소스에 `//Todo. 삭제`). 지우면 구 200 → 신 404 로 갈리고,
   지금 라우트가 `LEGACY` 라 **이미 열려 있는 경로**라 신에서 빼도 노출은 안 줄어든다.
   `JwtControllerTest` 가 계정 리터럴을 값으로 못박아 "조용히 다른 계정으로 바뀌는 것"만 막는다.
   **제거는 P6 이후 결정 항목으로 올린다.**
2. **구의 OTP 조회가 `type` 인자를 where 절에 안 쓴다**(`OtpRepositoryImpl`). `otp` 는 email 유니크라
   SIGNUP 으로 받은 코드로 LOGIN 이 통과한다. `usable` 인자도 버리고 리터럴 `true` 를 쓴다.
   구의 실동작이라 그대로 뒀다 — 고치면 "구는 통과, 신은 401" 방향이 된다. **보안 백로그.**
3. **`sendSmsSvc` 에는 `serviceMode` 가드가 없다**(알림톡에는 있다). 구는 개발계에서도 실제 NCP SENS 로
   문자를 쐈다. 코드에 가드를 새로 넣지 않고 `naverCloud.sesBaseUrl`·`aws.awsEamilServerHost` **기본값을
   스텁**으로 두어 봉인했다(저장소 기존 방식). **실키/실주소를 env 로 주는 순간 진짜 나간다** —
   P5 에서 OTP 경로를 리플레이하려면 스텁 고정 여부를 먼저 확인할 것.
4. **개발계 실키 적용은 P5 진입 조건이다.** `config/legacy.yml` 의 `medilawyer.jwt.{access,refresh}tokenSecretKey`
   기본값은 자리표시자다. 실키 없이 흡수하면 구 발급 토큰이 신에서 전부 401 이다.
   env 이름은 이미 열려 있다(`LEGACY_JWT_ACCESS_SECRET` / `LEGACY_JWT_REFRESH_SECRET`) —
   AC5 재현 명령은 `LegacyJwtCrossVerificationTest` javadoc 에 그대로 적어 뒀다(`--no-daemon` 필요).
5. **구 `token` 테이블에 현재 키로 안 열리는 행이 있다** — 2025-11-05 발급분이 `토큰 형식 오류`가 난다
   (키 회전 흔적). 흡수와 무관하지만, "구 발급 토큰이면 다 열린다"가 아니라는 뜻이라 P5 판정 시 유의.
6. **`/api/v1/otp/send` 는 리플레이로 검증할 수 없다.** OTP 코드가 매 요청 난수라 응답 `maskedPhone` 말고는
   비교할 것이 없고, 재생하면 메일·SMS 가 스텁으로라도 나간다. 계약 스모크(R9)로 분류할 것을 제안한다.
7. **중간에 전체 스위트가 45분간 멈춘 회차가 있었다** — 같은 머신에서 다른 워크트리 2개가 동시에
   Gradle 을 돌려 `modules:post` 의 Testcontainers PG 연결이 `doAuthentication` 에서 블록됐다(스레드덤프
   확인). 코드 문제가 아니라 환경 경합이고, 위 수치는 그 뒤 **한 번의 클린 전량 실행**으로 다시 잰 것이다.

STATUS: DONE
