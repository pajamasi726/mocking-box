# 12. fake-useragent 역할별 분리 + 개발계 병행 배포
이미지 `crawler-worker:dev-3role`=`68cebf79168c`(롤백 기준 `dev-3role-ua140` 유지). `roles/*`·원본 5저장소 무수정 — 원본 저장소의 기존 변경분은 mtime 전부 08-06 자로 이번 세션 수정 0건.

## 변경 파일
| 파일 | 내용 |
|---|---|
| `Dockerfile` | builder 에 `pip install --no-deps --target /opt/ua-compat fake-useragent==1.4.0` → runtime 으로 `COPY --from=builder --chown=10001:10001`. 140KB, 실측 내용물은 `fake_useragent`+dist-info 둘뿐이라 site-packages 를 가릴 위험 없음 |
| `docker/entrypoint.sh` | 역할별 `UA_COMPAT_DIR` 분기 → `PYTHONPATH="${ROLE_DIR}${UA_COMPAT_DIR:+:…}${PYTHONPATH:+:…}"`. 역할 디렉토리가 계속 최우선 |
| 문서 3종 | `requirements.txt` (b') "상향"→"역할별 분리" 정정 · `README.md` UA 절 전면 교체+개발계 병행 명시 · `docker-compose.cutover.yml` 머리에 "현재 병행·미사용"(삭제 안 함) |

## 역할별 실측 · UA 분포
| | thread | delegate | receipt |
|---|---|---|---|
| `__file__` · version | `/opt/ua-compat` · **1.4.0** | `/opt/ua-compat` · **1.4.0** | site-packages · **2.2.0** |
| `ua.random` 모바일 n=400 | **0 (0.0%)** | **0 (0.0%)** | 65.8% (receipt 는 안 쓰는 경로) |

`get_ua()` n=120 → 모바일 유출 **0** · non-Chrome **0** · Chrome 133~135 **96건(80%)**. chromium **140.0.7339.16** 실기동, `new_context(user_agent=)`↔`navigator.userAgent` **3/3 일치**·전부 데스크톱.
**무한루프 위험은 실질적으로 없다**: 2.2.0 `ua.chrome` 데스크톱 비율 **p=1381/3000=0.460** → 기대 반복 **2.17회(≈4.7ms)**. 100회 연속 모바일 확률 1.6e-27(그때 누적 대기 215ms). 위험해지는 조건은 **p=0**(풀 전체가 모바일) 하나뿐이라, 향후 상향 시 그 값만 확인하면 된다.

## 회귀 재검증 (전부 실행)
| 항목 | 결과 |
|---|---|
| healthy·`/health` | 신규 3/3 healthy·42130/42110 **200** / 기존 3/3 healthy·별칭 `crawler-web:42030`·`ai-delegate-web:42010` **200** |
| 별칭·Kafka 격리 | 별칭 3종 전부 **기존** 단일 IP(.22/.23/.30)·신규 별칭 **0건**. 브로커 실측 구 `…pg-receipt-fixture-v1`/`dev.fixture.*` ↔ 신 `crawler-worker-3role-verify-v1`/`test.3role.*` — 그룹·토픽 교집합 **0**, 각 멤버1·lag0, 구 오프셋 보존 |
| 공용망 차분 | 34→32건. 감소 2건 = 신규 thread/delegate 가 공용망에 미부착(dev.yml 단독), 기존 3건 exited→running. **running 28→29, 죽은 컨테이너 0** |
| 라우트 A/B | thread **14=14** · delegate **16=16** — 이미지·라이브 컨테이너 양쪽, route객체수/쌍/HEAD제외/고유path 4개 지표 전부 old==new |
| 스위트·`pip check`·healthcheck | delegate **43 passed** · receipt **4 passed** / 3역할 `No broken requirements found.` / receipt healthcheck 워커 있음 **exit=0**·없음 **exit=1** |
| import 감사 | thread 65:58/7 · delegate 84:81/3 · receipt 12:12/0 — 실패 모듈 집합이 `ua140` 이미지와 **문자열까지 동일, 회귀 0** |

## 정정 · 리뷰어에게
- 이전 문서 수치 3건 정정: **delegate 라우트 21→16**, **스위트 38→43**(38 은 `test_fixture_runtime.py` 제외분 — 그 파일이 import 하는 `fixture_runtime.py` 가 이미지 allowlist 밖이라 마운트 필요, `--ignore` 하면 38 재현), **import 감사 thread 57/8→58/7**. 셋 다 신·구 이미지가 서로 동일해 이번 변경의 회귀는 아니다.
- 남은 불일치 하나: thread/delegate 는 UA 가 데스크톱으로 돌아왔어도 하드코딩된 `sec-ch-ua`(Chrome122/Edge128)와 UA 계열이 어긋나는 조합은 남는다(FF 40%). **운영 원본과 동일한 동작**이고 `roles/*` 무수정 원칙에 걸려 건드리지 않았다.
- 개발계는 병행 상태다. 기존을 내리는 시점은 사용자 결정이며 그때 `docker-compose.cutover.yml` 을 얹는다. 커밋하지 않았다.
STATUS: DONE
