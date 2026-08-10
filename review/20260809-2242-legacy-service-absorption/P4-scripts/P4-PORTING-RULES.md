# P4 이식 규칙 (반드시 그대로 지킬 것)

구 저장소 = `/Users/steve/steve/legal-care/medilawyer-boot` (**절대 수정 금지 · 읽기 전용**)
신 저장소 = `/Users/steve/steve/legalcare-renew-prodsync-wt`
대상 모듈 = `apps/booster-app/modules/legacy/src/main/java/legalcare/medilawyer/legacy/`

## 0. 절대 금지

- **커밋·푸시 금지**
- **Gradle 실행 금지** (`./gradlew` 금지 — 다른 워크트리와 경합해 45분 행이 난 전례가 있다. 컴파일은 오케스트레이터가 중앙에서 한 번에 돌린다)
- `medilawyer-boot` 수정 금지 · 게이트웨이 수정 금지 · docker/컨테이너 조작 금지 · 외부 실호출 금지
- 지시된 파일 외 수정 금지

## 1. 1:1 이식 원칙

Kotlin → Java **1:1**. 클래스명·메서드명·필드명·필드 선언순서·URL 경로·에러 메시지 문자열·HTTP 상태를 **그대로** 보존한다.
**개선·리팩터링·계약 변경·버그 수정 금지.** 구의 버그도 그대로 옮긴다(주석으로 "구 그대로"라고 남긴다).

패키지 매핑: `legalcare.medilawyer.X` → `legalcare.medilawyer.legacy.X`

## 2. Kotlin → Java 변환 규칙

| Kotlin | Java |
|---|---|
| `data class Foo(val a: String, val b: Int?)` (응답 DTO) | `@Getter @JsonPropertyOrder({"a","b"})` + private final 필드 + 전필드 생성자 |
| `data class Foo(...)` (요청 바디 DTO) | `@Getter @Setter @NoArgsConstructor @JsonPropertyOrder({...})` + 필드 (아래 §3 non-null 규칙 적용) |
| `val x: T` (non-null, 기본값 없음) | §3 참조 |
| `val x: T?` | 그냥 nullable 필드 |
| `Int`/`Long`/`Boolean` non-null | 박싱 타입(`Integer`/`Long`/`Boolean`) — §3 이유 |
| `object Foo` | `public final class Foo` + private 생성자 + static |
| `companion object { const val X }` | `public static final X` |
| 최상위 `private val log = KotlinLogging.logger {}` | `@Slf4j` (lombok) — 로그 문자열도 그대로 |
| `enum class E(val code: Long, ...)` | `@Getter public enum E` + private final 필드 + 생성자 |
| `suspend fun` | 평범한 메서드 (구 Spring MVC 는 코루틴을 블로킹으로 돌린다 — 동작 동일) |
| `?.let { }` | null 체크 |
| `?:` | 삼항 / `Objects.requireNonNullElse` |
| `it` 람다 | 명시 파라미터명 |
| `mapOf(a to b)` | `Map.of` — **단 순서가 응답에 실리면 `LinkedHashMap`** |
| `listOf()` | `List.of()` (불변) |

## 3. **Kotlin non-null 소실 복원 — 최중요**

구 Kotlin 의 **기본값 없는 non-null 프로퍼티**는 `jackson-module-kotlin` 이 역직렬화 시
키 누락·명시적 null 둘 다에서 예외를 던졌다. Java 로 옮기면 조용히 `null` 이 되어
**구 4xx/5xx → 신 200** 이라는 최악의 방향으로 갈린다(P2 라운드1 BLOCKER).

**규칙**: 그 필드가 **역직렬화 경로**(= `@RequestBody` 로 받는 타입, 또는 외부 응답을
`.body(X.class)` 로 받는 타입, 또는 Kafka 컨슈머가 역직렬화하는 타입)에 있으면
`legalcare.medilawyer.legacy.common.jackson.LegacyNonNull` 을 필드에 붙인다.

- **직렬화 전용(응답)** 타입에는 **붙이지 않는다** (초과 부착 0 이 검증 대상이다)
- primitive 에 붙이면 기동 시 터진다 → 반드시 박싱 타입으로
- 중첩 타입도 같은 규칙

## 4. `@JsonPropertyOrder` 필수

`db/dto/**`·`common/response/**`·`common/exception/**` 아래 **인스턴스 필드를 가진 모든 구체 클래스**는
`@JsonPropertyOrder({...})` 를 달아야 하고 목록이 **필드 선언 순서와 정확히 일치**해야 한다.
(`JsonPropertyOrderCoverageTest` 가 전수 게이트로 잡는다. 중첩 static 클래스도 대상.)

## 5. 밑줄로 시작하는 필드

구 Kotlin `val _kau: String?` 같은 필드는 Java 게터가 `get_kau()` 가 되어 **Jackson 이 프로퍼티로 인식하지 못하고 통째로 증발**한다.
→ 반드시 `@JsonProperty("_kau")` 를 명시한다(요청·응답 양쪽).

## 6. 개인정보 로깅 금지

전화번호·실명·이메일·복호화된 채널 계정 ID·토큰·비밀번호를 로그에 찍지 않는다.
구가 찍고 있으면 **마스킹해서** 옮기고 주석으로 명시한다(이 저장소의 확립된 예외다 — `LegacyLogMasking` 참조).

## 7. 빈 이름 충돌 회피

booster 는 10개 모듈이 한 컨텍스트다. 다른 모듈과 단순명이 겹칠 수 있으므로:
- `@RestController("legacyXxxController")`, `@Repository("legacyXxxRepository")`,
  `@Component("legacyXxxImpl")`, `@Service("legacyXxxService")` 처럼 **`legacy` 접두 빈 이름**을 준다
  (기존 이식분 선례를 따를 것 — `TeamController.java` 는 `@RestController("legacyTeamController")`).
- 겹치지 않는 것이 확실해도 legacy 모듈 규칙상 접두를 붙인다.

## 8. 엔티티 규칙

- `extends BaseEntity`(구 `BaseEntity` 상속분) · `implements Persistable<T>`(구가 그랬다면)
- `@Getter` 만. `@Setter` 는 구가 `var` 인 필드에만, 그것도 구가 setter 를 쓰는 경우만.
  구가 `var` + 도메인 메서드로 바꾸면 **도메인 메서드를 그대로 옮긴다**.
- `protected Xxx() {}` 무인자 생성자 필수(JPA)
- ULID 기본값(`UlidCreator.getUlid().toString()`)이 구 필드 기본값이면 그대로
- `@Table(name=...)` 물리 테이블명 그대로
- 컬럼명·length·nullable·updatable 그대로
- **읽지 않는 연관을 임의로 생략하지 말 것.** 이번 슬라이스가 실제로 쓰면 반드시 옮긴다.

## 9. 주석

- 클래스 javadoc 첫 줄에 반드시 `구 {@code <구 상대경로>} 1:1 이식.` 형태로 원본 위치를 남긴다
- 비즈니스 로직 주석은 **한국어**
- 구와 달라진 점이 있으면 반드시 주석으로 근거를 남긴다

## 10. 참고할 기존 이식 선례 (읽어볼 것)

- 컨트롤러: `apis/http/team/controller/TeamController.java`
- 엔티티: `db/domain/message/Message.java`
- 리포지토리 3종: `db/domain/message/repository/*`
- DTO: `db/dto/reviewBoost/PresentHistoryByTeamRes.java`
- 예외: `common/exception/CustomException.java`, `common/exception/LegacyExceptionAdvice.java`
- non-null: `common/jackson/LegacyNonNull.java`, `LegacyNonNullModule.java`
