# mocking-box 배포 아키텍처 및 전체 그림 (v0.3 설계)

> 작성일: 2026-07-14 · 선행: [01 조사](01-differential-testing-tools.md) · [02 코어 설계](02-architecture.md)
> 대화에서 확정된 전체 구조의 박제. 핵심 원칙: **운영 무침투(box), 운영 무부하, 어떤 규모든 설치 당일 가치.**

## 1. 두 가지 검증 모드

| | **Record & Verify** (주력) | **Live Parallel** |
|---|---|---|
| 방식 | 수집기가 요청+**기대응답+기대 write-set**을 골든으로 저장 → 신서버에만 재생, 골든과 비교 | 같은 요청을 실시간으로 구·신 양쪽에, 즉석 비교 |
| 구서버 | 수집 때만 필요 | 항상 필요 |
| 반복성 | 골든 1개로 무한 반복 (CI 게이트) | 그 순간뿐 |
| 계보 | Keploy (단, SQL mock 대신 write-set 비교라 ORM 전환에도 유효) | Diffy / Zalando parallel-run |

두 모드는 diff 엔진·리포트·UI를 공유한다. v0.2까지는 Live만 구현, v0.3에서 R&V 추가.

## 2. 트래픽 수집 입력 소스 4종 (모두 같은 골든 포맷으로 수렴)

| 입력 | 침투도 | 조건 | 용도 |
|---|---|---|---|
| ① in-path 프록시 | 요청 경로 삽입 | 없음 | dev/스테이징, 데모 (v0.2 구현됨) |
| ② pcap 에이전트 (사이드카) | 경로 밖, 호스트에 컨테이너 1개 | 평문 홉 + CAP_NET_RAW | **운영 기본값.** tcpdump와 같은 libpcap — 차이는 TCP 재조립→HTTP 파싱→쌍 맺기 후처리 |
| ③ .pcap 파일 오프라인 변환 | 0 (tcpdump만 사용) | 서버에 tcpdump | 보수적 조직의 진입로: `tcpdump -w` 파일을 받아 로컬 변환 |
| ④ VPC Traffic Mirroring 수신 | **운영 호스트 제로터치** | AWS + Nitro 인스턴스 | AWS에서 가장 안전한 운영 수집 |

### 권한 설정 (②)
- VM: `sudo setcap cap_net_raw+eip mockingbox` (tcpdump 배포 방식과 동일)
- Docker: `docker run --network host --cap-add=NET_RAW ... agent`
- K8s: DaemonSet + `hostNetwork: true` + `capabilities.add: ["NET_RAW"]`
- promiscuous 불필요(자기 호스트 트래픽만). 미러 수신(④)일 때만 해당.

### VPC Traffic Mirroring 상세 (④)
- **왜 가장 안전한가**: 패킷 복사가 앱 인스턴스가 아니라 AWS 네트워크(Nitro) 계층에서 일어남.
  운영 호스트에 프로세스 0개. 혼잡 시 **AWS가 미러 트래픽부터 드랍**(운영 우선) — fail-open이 플랫폼 보장.
- 구성 요소: Source(게이트웨이 ENI) / Target(에이전트 EC2의 ENI 또는 NLB) / Filter(포트 규칙) / Session.
- 수신 측: VXLAN(UDP 4789) 캡슐이므로 에이전트가 decap 후 TCP 재조립 (gopacket VXLAN 레이어).
- 제약: source가 Nitro 인스턴스여야 함(t3/m5/c5 등 OK, **t2/m4 불가**), AWS 한정,
  미러 트래픽이 source ENI 대역폭에 계상(통상 무시 가능), 실데이터가 target으로 흐르므로 target 보안그룹 잠금(UDP 4789를 source에서만).
- 비용: 미러 세션 ENI당 시간당 소액(월 ~1만원대) + 에이전트 인스턴스.

Terraform (전부 코드화 가능):
```hcl
resource "aws_ec2_traffic_mirror_filter" "mb" { description = "mocking-box" }

resource "aws_ec2_traffic_mirror_filter_rule" "ingress" {
  traffic_mirror_filter_id = aws_ec2_traffic_mirror_filter.mb.id
  traffic_direction        = "ingress"
  rule_number              = 100
  rule_action              = "accept"
  protocol                 = 6
  source_cidr_block        = "0.0.0.0/0"
  destination_cidr_block   = "0.0.0.0/0"
  destination_port_range { from_port = 8080, to_port = 8080 }  # 대상 서비스 포트만
}
resource "aws_ec2_traffic_mirror_filter_rule" "egress" {  # 응답 방향
  traffic_mirror_filter_id = aws_ec2_traffic_mirror_filter.mb.id
  traffic_direction        = "egress"
  rule_number              = 100
  rule_action              = "accept"
  protocol                 = 6
  source_cidr_block        = "0.0.0.0/0"
  destination_cidr_block   = "0.0.0.0/0"
  source_port_range { from_port = 8080, to_port = 8080 }
}
resource "aws_ec2_traffic_mirror_target" "agent" {
  network_interface_id = aws_instance.mb_agent.primary_network_interface_id
}
resource "aws_ec2_traffic_mirror_session" "gw" {
  network_interface_id     = var.gateway_eni_id          # 소스: 게이트웨이 ENI
  traffic_mirror_target_id = aws_ec2_traffic_mirror_target.agent.id
  traffic_mirror_filter_id = aws_ec2_traffic_mirror_filter.mb.id
  session_number           = 1
}
```
콘솔 경로: VPC → Traffic Mirroring → (Filters / Targets / Sessions). 앱 배포·코드와 완전 무관한 네트워크 리소스라 언제든 세션 삭제로 원복.

## 3. 골든 아티팩트 (포터블 단일 파일)

```
name.golden.jsonl
  {"type":"meta", "version":1, "created_at":..., "upstream":..., "serialized":true}
  {"type":"entry", "name":..., "method":..., "path":..., "headers":..., "body":...,
   "expected": {"status":200, "body":"...", "writeset":[...] | null}}
```
- `expected.writeset`은 **귀속 가능할 때만** 채움: 순차화된 프록시 수집(dev) 또는 Level 1(rid 태깅).
  운영 동시 트래픽 수집(pcap/미러)에서는 요청 단위 귀속이 불가하므로 null — 이때 write 검증은
  수집창 전체 binlog 집계 vs 재생 전체 write-set 집계 비교(총합 모드, v0.4) 또는 응답 diff만.
- 스냅샷(시딩용 덤프)은 골든과 짝을 이루는 별도 파일(`name.snapshot.sql.gz`) — v0.4에서 자동화.

## 4. 상태(DB) 정합 원칙

- **트래픽과 DB 상태는 같은 환경·같은 시점(T0)이어야 한다.** 운영 트래픽 + 개발 DB = 무효.
- 신서버는 **반드시 별도 DB** (같은 DB면 쓰기 이중 적용). "별도 DB" = 빈 MySQL 컨테이너 하나 + 시딩.
- 시딩은 전체가 아니라 **접촉 테이블만** (write-set이 목록을 알려줌 — 수직 서브셋).
  이미지/블롭 테이블은 검증 대상 write 경로에 없으므로 복사 대상이 아님.
- 시딩 소스 사다리 (운영 부하 0 우선):
  1. dev/스테이징 DB (환경 정합 주의 — 트래픽도 dev일 때만)
  2. read replica에서 덤프
  3. **기존 백업 + PITR**: RDS는 자동 백업만 켜져 있으면 T0 시점 임시 인스턴스 복원(운영 무접촉)
     → 접촉 테이블 덤프 → 임시 인스턴스 삭제
  4. 스토리지 스냅샷: Aurora fast clone(CoW), RDS 스냅샷, 온프렘 ZFS
  5. (최후) primary에서 `--single-transaction` 직접 덤프
- 반복 리셋 비용은 DB 크기가 아니라 쓰기량에 비례하게: 시딩 1회 후 재실행은 flashback 역실행 or CoW 재클론.

## 5. write-set 캡처의 실체 (무침투)

- binlog replication client = **TCP 접속 1개 + 계정 1개** (`GRANT REPLICATION SLAVE, REPLICATION CLIENT`).
  서버 설치·재시작 없음. 부하는 binlog 파일 순차 읽기+네트워크 전송(레플리카 1대 수준, Debezium/DMS 표준 경로).
- RDS 체크리스트: ① 자동 백업 ON(binlog 활성 전제, PITR 전제와 동일) ② 파라미터 그룹 `binlog_format=ROW`(동적)
  ③ `CALL mysql.rds_set_configuration('binlog retention hours', 24)`
- DB 다양성: WriteSetSource 커넥터 구조 — MySQL binlog(구현) → PostgreSQL logical decoding(pglogrepl) → Kafka 토픽 tap(발행 이벤트 검증) → Mongo change streams. 커넥터 없는 저장소는 응답 diff로 degrade.
- 스택당 소스 복수(`sources: [...]`) — legalcare처럼 MySQL+PostgreSQL 동시 사용 대응.

## 6. 수집기 저장 원칙 (호스트 디스크 보호)

- 세그먼트 롤링 스트리밍: 64MB/5분 세그먼트 → S3 업로드 → 삭제. **로컬 점유 상한 ~128MB.**
- 스냅샷 덤프는 `mysqldump | gzip | S3` 파이프 (디스크 무경유).
- **fail-open**: 녹화는 비동기 사이드 작업. 저장 실패·버퍼 초과 시 드랍 카운트만 남기고 실트래픽은 무조건 통과.
- 골든 이동: ① S3 버킷 = 골든 저장소 (EC2는 IAM Role이라 자격증명 무입력) ② 원격 박스 API 직결(토큰) ③ 수동 파일 복사. SSH를 UI가 요구하는 기능은 만들지 않음.

## 7. legalcare 배치도

```
[AWS 운영 VPC]
  ALB(TLS 종료) → gateway → 서비스들 → RDS MySQL(1대, 레플리카 없음)
        │
        ├─ 수집: VPC Traffic Mirroring(게이트웨이 ENI → 에이전트 EC2)  ← 권장
        │        또는 pcap 사이드카 (게이트웨이 EC2에 컨테이너)
        ├─ write-set: 에이전트가 RDS binlog 구독 (접속 1개)
        └─ 시딩: RDS PITR → T0 임시 인스턴스 → 접촉 테이블 덤프 → 삭제
        ↓ (전부 S3로 스트리밍)
[S3]   goldens/…golden.jsonl + snapshot.sql.gz
        ↓
[DELL] mockingbox 콘솔: 골든 다운로드 → 신DB(빈 컨테이너) 시딩 → renew에 재생 → 리포트 반복
```

## 8. 대규모(카카오/라인급) 대응 포지션

- 검증 단위는 항상 "서비스 1개" — 전사 2벌은 아무도 안 하고 필요도 없음.
- 대규모 live는 in-path 금지: **mesh 미러(Envoy/Istio)의 tap을 받는 comparator 모드** + 샘플링 + 비동기 비교.
- 남는 큰 조각: **egress 부수효과 흡수/mock** (candidate의 아웃바운드 콜 차단·기록) — 로드맵 필수 항목.
- 쓰기 live 검증은 CDC 샌드박스(Signadot 모델) 영역 — R&V로 위임하는 것이 우리 포지션.

## 9. 구현 로드맵 (갱신)

| 단계 | 내용 |
|---|---|
| **v0.3 (지금)** | 골든 포맷, R&V 모드(프록시 수집이 응답+write-set 골든화, 순차화 옵션), verify 실행(신서버만), UI 반영 |
| v0.4 | pcap 에이전트 + .pcap 변환, S3 골든 저장소 + 설정 위저드, PITR/mysqldump 시딩 자동화, 집계 write 검증 |
| v0.5 | VPC 미러 수신(VXLAN), 스택당 복수 소스(PostgreSQL 커넥터), Kafka tap |
| v0.6 | Level 1 귀속(SQLCommenter), egress 기록/mock, tap comparator(대규모 live) |
