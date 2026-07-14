# demo-pg — PostgreSQL 검증 데모 (medilawyer 축소판)

legalcare의 실제 마이그레이션(jOOQ→JPA, PostgreSQL medilawyer-prod)을 본뜬 PG 검증 시나리오.
MySQL 데모(`demo-legalcare`)와 동일한 구조를 PostgreSQL로 재현 — **PG 커넥터 검증용**.

| 실제 | 시뮬레이션 |
|---|---|
| medilawyer PostgreSQL 16 (:15431, wal_level=logical) | pg-old/pg-new (:15432/:15433, wal_level=logical) |
| crawling-service (jOOQ, 부스터 심장) | app-old :10203 (상대 UPDATE, ORDER BY ASC) |
| 리라이트 (JPA) | app-new :10204 (FOR UPDATE + 절대 UPDATE, 버그 내장) |
| booster 스키마 (리뷰 도메인) | booster.review / review_history |

신스택 심어둔 것: ① publish 시 review_history INSERT 누락(응답 동일 → write-set만 검출)
② 목록 ORDER BY DESC(매퍼 차이 → sort_arrays 흡수).

## 커넥터 검증 항목 (전부 통과)

- **PG 탐색**: pg_class로 스키마·테이블·행수·크기 (Copy DB 체크박스)
- **PG 복사**: pure pgx — 스키마·테이블(컬럼·PK·default·serial 시퀀스)·데이터 COPY, 구→신
- **PG write-set**: logical decoding(pgoutput, proto v2) — 임시 publication+slot,
  REPLICA IDENTITY FULL로 완전 before-image, 커밋 LSN 기준 결정적 윈도우 귀속
- **PG 헬스**: 접속 + wal_level=logical 검사

## 실행

```bash
cd demo-pg && docker compose up -d --build

# 1) 골든 수집 (구 PG 앞 프록시, logical decoding write-set 포함)
mockingbox dashboard -c config.yaml --addr :8644   # 수집 탭에서 프록시 시작 or API
# 2) 신 PG를 구 상태로 복사 (DB 복사 탭 [구→신 복사])
# 3) 검증 (신서버 단독)
mockingbox verify -c config.yaml --golden corpus/pg-scenario.golden.jsonl
```

## 실측 결과 (2026-07-14)

```
1  GET /review/1            MATCH
2  POST /review/1/publish   WRITESET_DIFF  2/1  review_history[INSERT] → <absent>
3  POST /review/3/publish   WRITESET_DIFF  2/1  review_history[INSERT] → <absent>
4  GET /reviews             MATCH  (ORDER BY 차이는 sort_arrays로 흡수)
5  GET /review/1            MATCH
Total 5   MATCH=3  WRITESET_DIFF=2   ← 응답 동일한 write 버그를 PG logical decoding으로 검출
```

## legalcare 실전 적용 시 확인

- medilawyer RDS는 `wal_level=logical`이어야 함 (파라미터 그룹, 재부팅 필요) — write-set 캡처 전제
- `boost_dental_image`(975GB) 같은 대용량 테이블은 Copy DB 탭에서 체크 해제(제외)
- REPLICA IDENTITY FULL은 스키마 스코프 테이블에만 적용(메타데이터 변경) — 운영 영향 미미하나 사전 협의 권장
- DB 이름에 하이픈(`medilawyer-prod`)이 있으면 config `database:` 에 그대로 기입
