# 10. fake-useragent 1.4.0 → 2.2.0 (운영 UA 보존) + 재검증

사용자 결정("운영 그대로 유지") 반영. 이미지 태그는 `crawler-worker:dev-3role` 유지(compose 무수정),
직전 1.4.0 이미지는 `crawler-worker:dev-3role-ua140`(`f1c10c1d5440`)로 태그해 롤백·A/B 근거로 남겼다.
신규 이미지 `6a62cb00b2e7`. `roles/*` 무수정 — 서브셋 sha256 thread 72 · delegate 108 · receipt 17, **불일치 0**.

## 변경 파일
| 파일 | 내용 |
|---|---|
| `requirements.txt` | `fake-useragent==1.4.0`→`2.2.0`. 머리 주석에 `(b')` 절 신설(상향 사유=운영 동작 보존), `(c)` 하위핀 유지 목록 12→**11건**(fake-useragent 제거). 나머지 11건 무수정 |
| `README.md` | 의존성 표에 "운영 동작 보존을 위한 상향" 행 추가 · "낮아진 12건"→11건 · "결과가 다른 3건"→2건(fake-useragent 행 제거) · **"UA" 절 신설**(A/B 표 + thread/delegate 한계) · 미검증 표 행 교체 |

## 전제 정정 — receipt 는 모바일 UA 를 내보내지 않는다

지시서의 "운영 그대로 유지(모바일 135)"는 `ua.chrome` 을 직접 잰 값이다. 실제 receipt 경로는
`get_ua()`(`roles/receipt/crawlers/crawling_utils.py:23-31`)이고, **`Mobile`·`Android`·`iPhone` 이 든 값을
버리고 다시 뽑는 루프**다. 표본 100에서 모바일 유출 0건. 운영이 보내는 값은 **데스크톱 Chrome/135** 다.
결정(2.2.0 채택) 자체는 그대로 옳다 — 운영과 같은 풀을 복원한다.

## UA 분포 A/B (표본 100, `random.seed(42)`)

| 항목 | 1.4.0 | 2.2.0 | 평가 |
|---|---|---|---|
| **receipt** `get_ua()` 모바일 유출 | 0 | 0 | 동일(필터) |
| **receipt** `get_ua()` 계열 | Chrome 80 · **Opera 20** | Chrome **100** | 개선 |
| **receipt** `get_ua()` Chrome 메이저 | 77~117 (135 **0건**) | 109~135 (**135 가 57건**) | 운영 복원 |
| **receipt** `get_ua()` 빌드번호 비정상 | 3 | 1 | 개선 |
| thread/delegate `ua.random` 모바일 | **0** | **68** (iOS 53 · Android 15) | **악화** |
| thread/delegate `ua.random` 계열 | Chrome 39 · FF 28 · Edge 21 · Safari 9 · Opera 3 | Safari 49 · Chrome 34 · Edge 7 · CriOS 5 · 기타 5 | 변동 |
| thread/delegate `ua.random` Chrome <120 | 63 | 7 | 개선 |
| thread/delegate `ua.random` 빌드번호 비정상 | 1 | **11** | 악화 |

`ua.random` 예외 없이 동작(2.2.0 API 호환 OK). 이상 버전(`Chrome/42.0.6651.1925` 등)·`CriOS` 는 실재하나
**receipt 에는 도달하지 않고** thread/delegate HTTP 헤더에만 실린다.

## 재검증 (전부 재실행)

| 항목 | 결과 |
|---|---|
| 3역할 healthy | thread·delegate·receipt 3/3 healthy (10초 내) |
| 별칭 3종 | `crawler-web`→172.28.0.21 · `ai-delegate-web`→172.28.0.23 · `pg-receipt-worker`→172.28.0.22, **각 1개씩만**, 신규 컨테이너 IP와 일치. 구 3컨테이너 exited |
| `/health` | crawler-web **200** · ai-delegate-web **200** |
| 공용망 `legalcare-local_legalcare` | 교체 전 28 전부 running → 교체 후 28 전부 running, **상태 차분 0** |
| 라우트 A/B | thread old 14 = new 14 · delegate old 21 = new 21, **0 diff** (원본 이미지 대조) |
| 스위트 | delegate **38 passed** · receipt **4 passed** (tests 는 이미지 allowlist 밖이라 마운트 실행) |
| `pip check` | 신규·롤백 이미지 모두 `No broken requirements found.` |
| import 감사 | thread 65모듈 57/8 · delegate 84모듈 81/3 · receipt 12모듈 12/0 — **실패 모듈 집합이 1.4.0 이미지와 완전 동일, 회귀 0** |
| receipt healthcheck | 워커 있음 **exit=0** / 워커 없는 컨테이너 **exit=1** — 1라운드 수정 유지 |
| chromium + UA | chromium **140.0.7339.16** 기동, `get_ua()`→`new_context`→`navigator.userAgent` **3/3 MATCH**, 전부 데스크톱, Chrome/135 우세(3회 중 2회 135, 1회 114) |

## 한계

1. **[신규·주의] thread/delegate 헤더가 자기모순이 될 수 있다.** `ua.random` 은 `config.py` **최상위 dict
   리터럴**이라 프로세스당 1회만 뽑혀 재기동 전까지 고정된다. 그런데 9개 헤더 dict 중 **7개가
   `sec-ch-ua-mobile: "?0"` + `sec-ch-ua-platform: "Windows"` 를 하드코딩**한다(thread `config.py:215·230·248·
   263·276·322·421`, delegate `config.py:227·242·260·275·288·334·433`). 1.4.0 은 UA 가 항상 데스크톱이라
   일관됐지만, 2.2.0 에서는 **"UA 는 아이폰인데 client-hint 는 윈도우"** 조합이 프로세스당 약 68% 확률로 굳는다.
   봇 판별 신호로 쓰기 좋다. `roles/*` 무수정 원칙(R1) 때문에 코드로 막지 않았다 — 필터링·고정 UA 가 필요하면
   원본 저장소(`crawler`·`ai-delegate-crawler`)에서 다룰 별도 변경이다.
   개발계는 `ENABLE_EXTERNAL_ACTIONS=false` 라 이 헤더가 실사이트로 나가지 않으므로 지금 장애 요인은 아니다.
2. **네이버·카카오·모두닥이 이 조합을 어떻게 취급하는지 미검증** — 실접속 금지 제약. README 미검증 표에 행 추가.
3. `get_ua()` 는 모바일이면 무한 재시도한다. 2.2.0 `ua.chrome` 의 데스크톱 비율이 56%라 현재는 안전하지만,
   풀이 전부 모바일인 버전으로 올리면 무한루프가 된다. 향후 fake-useragent 상향 시 확인 필요.
4. receipt UA 는 여전히 **비결정적**이다(2.2.0 에서도 135 는 57%, 109까지 내려감). 운영과 같은 성질이라
   회귀는 아니지만 "항상 135" 가 아니라는 점은 기록해 둔다.
5. 09-final-report 의 잔여 리스크 2·4·5·6(구 compose 정의 · 운영 배포 경로 · 이미지 3.14GB · 검증용 토픽)은 이번 범위 밖으로 그대로 남아 있다.

STATUS: DONE
