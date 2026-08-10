# 사전조사: legacy-service → booster-app 흡수 (사용자 지시 2026-08-09)

> "다 하고 나면 레거시 서비스도 부스터 api 로 합쳐버려 그리고 개발계에서 전체 테스트해 자체 리뷰로"

## 정체 확인
`legacy-service` = **`/Users/steve/steve/legal-care/medilawyer-boot`** (구 코틀린 모놀리스)
- 근거: `module-core/core-api/src/main/resources/application.yml:11` → `spring.application.name: legacy-service`
- 배포 이미지명도 `legacy-service-{profile}` (`build.gradle.kts:109`)
- git HEAD `930c83cf` (Merge branch 'develop')

## 규모
| 항목 | 값 |
|---|---|
| 코틀린/자바 파일 | 638 |
| 소스 줄수 | 28,861 |
| `@RestController` | 38 |
| 모듈 | module-client · module-core · module-storage · module-support |

## 이미 확보한 실측 (재조사 불필요)
- **레거시 `/api/v1` 147개 중 76개가 실사용**(APM 3개월). 미사용 71개.
  76개 중 **20개는 3개월 통틀어 1~9회** — 관리자·초기설정 기능이라 짧은 관측 창에서는 안 보인다.
- **legacy-service 는 크롤러를 HTTP 로 부르는 유일한 서비스다**(운영 90일 1,075건 = 하루 12건).
  → 흡수하면 `handler.crawler.url`(운영 공인IP 하드코딩)·`prod-delegate.medilawyer.co.kr` 호출 주체가 booster 로 옮겨온다.
  주소 정리(운영 크롤러 DNS 이름 부여)를 이 작업과 묶어 처리할 수 있다.
- 크롤러 호출 지점: `module-client/core-client/.../ml/service/MlService.kt`(reply_gpt_return·kakao_login_2·naver_login_captcha),
  `module-client/core-client/.../external/ExternalClientService.kt`(get_naver_place_list)
- 게이트웨이에 `legacy-v2` 라우트(`/api/v2` → LEGACY) 가 이미 있음 (`Routes.java`)

## 흡수 대상 저장소
`/Users/steve/steve/legalcare-renew/apps/booster-app` (Spring Boot 4 / Java 21 멀티모듈)
