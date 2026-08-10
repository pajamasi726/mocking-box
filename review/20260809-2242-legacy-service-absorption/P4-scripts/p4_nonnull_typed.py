#!/usr/bin/env python3
"""p4_nonnull.py 의 타입 인지형 후속 — 라운드1-P4 B1.

구 저장소가 있어야 돌기 때문에 CI 에는 못 넣는다. 빌드 안의 게이트는
modules/legacy/src/test/.../LegacyNonNullInventoryTest 이고, 이 스크립트는 그 게이트가
쓰는 동결 목록(nonnull-inventory.tsv)을 만들고 구 원본과 대조하는 오프라인 도구다.

B1 — @LegacyNonNull 부착 지점을 구 Kotlin 타입 기준으로 원시형/참조형 분류.

구 jackson-module-kotlin 은 기본값 없는 non-null 이라도
  - 참조형 -> MissingKotlinParameterException
  - 원시형 -> 안 던짐. Jackson PropertyValueBuffer._findMissing 이 JVM 기본값(0/false) 주입
이라 두 부류를 갈라야 한다.
"""
import os, re, sys, json

OLD = "/Users/steve/steve/legal-care/medilawyer-boot"
NEW = "/Users/steve/steve/legalcare-renew-prodsync-wt/apps/booster-app/modules/legacy/src/main/java/legalcare/medilawyer/legacy"

KT_PRIMITIVE = {"Long", "Int", "Boolean", "Double", "Float", "Short", "Byte", "Char"}


def bal(src, start, op='(', cl=')'):
    d = 0; i = start; n = len(src)
    while i < n:
        c = src[i]
        if c == '"':
            if src[i:i+3] == '"""':
                j = src.find('"""', i+3); i = (j+3) if j != -1 else n; continue
            i += 1
            while i < n:
                if src[i] == '\\': i += 2; continue
                if src[i] == '"': i += 1; break
                i += 1
            continue
        if c == op: d += 1
        elif c == cl:
            d -= 1
            if d == 0: return i
        i += 1
    return -1


def split_top(body):
    parts = []; depth = 0; cur = ""; i = 0
    while i < len(body):
        ch = body[i]
        if ch == '"':
            j = i+1
            while j < len(body):
                if body[j] == '\\': j += 2; continue
                if body[j] == '"': j += 1; break
                j += 1
            cur += body[i:j]; i = j; continue
        if ch in '([<{': depth += 1
        if ch in ')]>}': depth -= 1
        if ch == ',' and depth == 0: parts.append(cur); cur = ""
        else: cur += ch
        i += 1
    if cur.strip(): parts.append(cur)
    return parts


# ---------- 구: pkg.Class.field -> (kotlin type, has_default, file) ----------
old_props = {}
for dp, dn, fn in os.walk(OLD):
    if "/build/" in dp or "/src/test/" in dp: continue
    for f in fn:
        if not f.endswith(".kt"): continue
        p = os.path.join(dp, f)
        src = open(p, encoding="utf-8", errors="replace").read()
        src = re.sub(r'/\*.*?\*/', '', src, flags=re.S)
        src = re.sub(r'//[^\n]*', '', src)
        pm = re.search(r'^package\s+([\w.]+)', src, re.M)
        pkg = pm.group(1).replace("legalcare.medilawyer.", "") if pm else ""
        # 중첩 클래스는 Outer.Inner 로 키를 만든다(같은 패키지에 CookieDict 4벌 등 단순명 충돌)
        kt_classes = []
        for m in re.finditer(r'\b(?:data\s+)?class\s+(\w+)\b', src):
            ob = src.find('{', m.end())
            if ob == -1: continue
            e2 = bal(src, ob, '{', '}')
            if e2 == -1: continue
            kt_classes.append((ob, e2, m.group(1)))

        def kt_owner(name, off):
            outers = [cn for ob, e2, cn in kt_classes if ob < off < e2 and cn != name]
            return ".".join(outers + [name])

        for m in re.finditer(r'\b(?:data\s+)?class\s+(\w+)\s*(?:<[^>]*>\s*)?\(', src):
            name = kt_owner(m.group(1), m.start())
            op = src.index('(', m.end()-1)
            e = bal(src, op)
            if e == -1: continue
            for part in split_top(src[op+1:e]):
                t = part.strip()
                fm = re.search(r'\b(?:val|var)\s+`?(\w+)`?\s*:\s*(.+)$', t, re.S)
                if not fm: continue
                fname = fm.group(1)
                rest = fm.group(2)
                # 기본값 판정: 최상위 '=' 존재 여부
                depth = 0; eq = -1
                for i, ch in enumerate(rest):
                    if ch in '([<{': depth += 1
                    elif ch in ')]>}': depth -= 1
                    elif ch == '=' and depth == 0 and rest[i:i+2] != '==':
                        eq = i; break
                has_default = eq != -1
                typ = (rest[:eq] if has_default else rest).strip()
                key = f"{pkg}.{name}"
                old_props.setdefault(key, {})
                # 같은 단순명이 여러 파일에 있으면 첫 번째 유지, 충돌은 리스트로
                old_props[key].setdefault(fname, []).append(
                    (typ, has_default, os.path.relpath(p, OLD)))

# ---------- 신: @LegacyNonNull / @LegacyNonNullPrimitive 부착 지점 ----------
rows = []
for dp, dn, fn in os.walk(NEW):
    if "/build/" in dp: continue
    for f in fn:
        if not f.endswith(".java"): continue
        p = os.path.join(dp, f)
        raw = open(p, encoding="utf-8", errors="replace").read()
        lines = raw.split("\n")
        src = re.sub(r'/\*.*?\*/', lambda m: "\n"*m.group(0).count("\n"), raw, flags=re.S)
        src = re.sub(r'//[^\n]*', '', src)
        pm = re.search(r'^package\s+([\w.]+)', src, re.M)
        pkg = pm.group(1).replace("legalcare.medilawyer.legacy.", "") if pm else ""
        # 클래스 경계(최상위 + 중첩) 추적: 필드 오프셋으로 소속 클래스 결정
        classes = []
        for m in re.finditer(r'\b(?:public\s+|protected\s+|private\s+|static\s+|final\s+|abstract\s+)*class\s+(\w+)', src):
            ob = src.find('{', m.end())
            if ob == -1: continue
            e = bal(src, ob, '{', '}')
            if e == -1: continue
            classes.append((ob, e, m.group(1)))
        for fm in re.finditer(
                r'((?:@\w+(?:\((?:[^()]|\([^()]*\))*\))?\s*)*)private\s+(?!static)(?:final\s+)?'
                r'([\w.$]+(?:\s*<[^;=]*>)?(?:\[\])?)\s+(\w+)\s*[;=]', src):
            anns = fm.group(1)
            jtype = re.sub(r'\s+', '', fm.group(2))
            fname = fm.group(3)
            marker = None
            if "@LegacyNonNullPrimitive" in anns: marker = "LegacyNonNullPrimitive"
            elif "@LegacyNonNull" in anns: marker = "LegacyNonNull"
            if not marker: continue
            off = fm.start(3)
            owners = [cn for ob, e, cn in classes if ob < off < e]
            owner = ".".join(owners)
            line = src[:off].count("\n") + 1
            key = f"{pkg}.{owner}"
            cand = old_props.get(key, {}).get(fname)
            if cand is None:
                # 단순명만으로 재탐색
                alt = [(k, v[fname]) for k, v in old_props.items()
                       if k.rsplit(".", 1)[-1] == owner and fname in v]
                cand = alt[0][1] if alt else None
                if alt: key = alt[0][0] + "(byname)"
            rows.append(dict(file=os.path.relpath(p, NEW), line=line, cls=key,
                             field=fname, jtype=jtype, marker=marker,
                             old=cand))

BOXED = {"Long": "Long", "Integer": "Int", "Boolean": "Boolean", "Double": "Double",
         "Float": "Float", "Short": "Short", "Byte": "Byte", "Character": "Char"}

prim, ref, unknown, mismatch = [], [], [], []
for r in rows:
    old = r["old"]
    if old is None:
        unknown.append(r); continue
    types = {t for t, d, f in old}
    defs = {d for t, d, f in old}
    r["old_type"] = "|".join(sorted(types))
    r["old_files"] = sorted({f for t, d, f in old})
    r["old_default"] = any(defs)
    kt = sorted(types)[0]
    ktbase = kt.rstrip("?")
    if kt.endswith("?"):
        mismatch.append((r, "구가 nullable 인데 마커가 붙어 있다"))
        continue
    if r["old_default"]:
        mismatch.append((r, "구에 기본값이 있다(=optional)"))
        continue
    if ktbase in KT_PRIMITIVE:
        r["expect"] = "PRIMITIVE"
        prim.append(r)
    else:
        r["expect"] = "REFERENCE"
        ref.append(r)

print(f"총 부착 지점 {len(rows)}")
print(f"  구=원시형(반드시 통과, JVM 기본값)  : {len(prim)}")
print(f"  구=참조형(반드시 던짐)              : {len(ref)}")
print(f"  구 대응 못 찾음                     : {len(unknown)}")
print(f"  기타 이상                           : {len(mismatch)}")
print()
print("=== 원시형(수정 대상) ===")
for r in sorted(prim, key=lambda x: (x["file"], x["line"])):
    ok = "OK" if r["marker"] == "LegacyNonNullPrimitive" else "**FIX**"
    print(f"{ok}\t{r['file']}:{r['line']}\t{r['jtype']} {r['field']}\t구={r['old_type']}\t{r['marker']}")
print()
print("=== 참조형인데 Primitive 마커가 붙은 것 ===")
for r in sorted(ref, key=lambda x: (x["file"], x["line"])):
    if r["marker"] == "LegacyNonNullPrimitive":
        print(f"**FIX**\t{r['file']}:{r['line']}\t{r['jtype']} {r['field']}\t구={r['old_type']}")
print()
print("=== 구 대응 못 찾음 ===")
for r in unknown:
    print(f"?\t{r['file']}:{r['line']}\t{r['jtype']} {r['field']}\t{r['cls']}\t{r['marker']}")
print()
print("=== 기타 이상 ===")
for r, why in mismatch:
    print(f"!\t{r['file']}:{r['line']}\t{r['jtype']} {r['field']}\t구={'|'.join(sorted({t for t,d,f in r['old']}))}\t{why}")

# 빌드 게이트가 읽는 동결 목록 생성 (인자로 경로를 주면 그리로 쓴다)
if len(sys.argv) > 1:
    lines = []
    for r in rows:
        parts = r["cls"].split(".")
        i = len(parts) - 1
        while i > 0 and parts[i - 1][:1].isupper():
            i -= 1
        fqcn = ("legalcare.medilawyer.legacy." + ".".join(parts[:i]) + "."
                + "$".join(parts[i:]))
        lines.append(f"{fqcn}\t{r['field']}\t{r['marker']}")
    with open(sys.argv[1], "w") as f:
        f.write("\n".join(sorted(lines)) + "\n")
    print(f"\n동결 목록 {len(lines)}행 -> {sys.argv[1]}")
