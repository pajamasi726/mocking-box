#!/usr/bin/env python3
"""[대체됨 — 라운드1-P4 B1] 이 스크립트는 타입을 보지 않아 구 계약을 뒤집는 부착을 통과시켰다.
구 jackson-module-kotlin 은 "기본값 없는 non-null" 이라도 원시형이면 던지지 않고 0/false 로
통과시킨다. 그 38곳을 이 스크립트가 정상으로 판정했다.
→ 타입 인지형은 p4_nonnull_typed.py, 빌드 안 게이트는 LegacyNonNullInventoryTest 다.

구 Kotlin '기본값 없는 non-null' 전수 추출 ↔ 신 @LegacyNonNull 부착 대조.
- 부착된 클래스: 집합이 정확히 같아야 한다(초과·누락 0)
- 미부착 클래스: 목록으로 뽑아 직렬화 전용인지 사람이 판정(문서에 근거를 남긴다)"""
import os,re,sys,collections
OLD="/Users/steve/steve/legal-care/medilawyer-boot"
NEW="/Users/steve/steve/legalcare-renew-prodsync-wt/apps/booster-app/modules/legacy/src/main/java/legalcare/medilawyer/legacy"

def bal(src,start,op='(',cl=')'):
    d=0;i=start;n=len(src)
    while i<n:
        c=src[i]
        if c=='"':
            if src[i:i+3]=='"""':
                j=src.find('"""',i+3); i=(j+3) if j!=-1 else n; continue
            i+=1
            while i<n:
                if src[i]=='\\': i+=2; continue
                if src[i]=='"': i+=1; break
                i+=1
            continue
        if c==op: d+=1
        elif c==cl:
            d-=1
            if d==0: return i
        i+=1
    return -1

def split_top(body):
    parts=[];depth=0;cur="";i=0
    while i<len(body):
        ch=body[i]
        if ch=='"':
            j=i+1
            while j<len(body):
                if body[j]=='\\': j+=2; continue
                if body[j]=='"': j+=1; break
                j+=1
            cur+=body[i:j]; i=j; continue
        if ch in '([<{': depth+=1
        if ch in ')]>}': depth-=1
        if ch==',' and depth==0: parts.append(cur); cur=""
        else: cur+=ch
        i+=1
    if cur.strip(): parts.append(cur)
    return parts

# 구: pkg.Class -> [기본값 없는 non-null 필드]
old_nonnull={}
for dp,dn,fn in os.walk(OLD):
    if "/build/" in dp or "/src/test/" in dp: continue
    for f in fn:
        if not f.endswith(".kt"): continue
        p=os.path.join(dp,f)
        src=open(p,encoding="utf-8",errors="replace").read()
        src=re.sub(r'/\*.*?\*/','',src,flags=re.S); src=re.sub(r'//[^\n]*','',src)
        pm=re.search(r'^package\s+([\w.]+)',src,re.M)
        pkg=pm.group(1).replace("legalcare.medilawyer.","") if pm else ""
        for m in re.finditer(r'\b(?:data\s+)?class\s+(\w+)\s*(?:<[^>]*>\s*)?\(', src):
            name=m.group(1); op=src.index('(',m.end()-1); e=bal(src,op)
            if e==-1: continue
            nn=[]
            for part in split_top(src[op+1:e]):
                fm=re.search(r'\b(?:val|var)\s+`?(\w+)`?\s*:\s*([^=]+)$', part.strip(), re.S)
                if not fm:
                    fm2=re.search(r'\b(?:val|var)\s+`?(\w+)`?\s*:\s*(.+)', part.strip(), re.S)
                    if not fm2: continue
                    continue   # '=' 있음 → 기본값 있음 → 대상 아님
                typ=fm.group(2).strip()
                if typ.endswith("?"): continue
                nn.append(fm.group(1))
            if nn: old_nonnull.setdefault(pkg+"."+name,[]).append((nn,os.path.relpath(p,OLD)))

# 신: pkg.Class -> [@LegacyNonNull 붙은 필드]  /  전체 필드
FIELD=re.compile(r'(?:^|\n)([^\n]*\n)?\s*private\s+(?!static)(?:final\s+)?[\w.$]+(?:\s*<[^;=]*>)?(?:\[\])?\s+(\w+)\s*[;=]')
new_marked={}; new_all={}; srcfile={}
for dp,dn,fn in os.walk(NEW):
    if "/build/" in dp: continue
    for f in fn:
        if not f.endswith(".java"): continue
        p=os.path.join(dp,f)
        raw=open(p,encoding="utf-8",errors="replace").read()
        src=re.sub(r'/\*.*?\*/','',raw,flags=re.S); src=re.sub(r'//[^\n]*','',src)
        pm=re.search(r'^package\s+([\w.]+)',src,re.M)
        pkg=pm.group(1).replace("legalcare.medilawyer.legacy.","") if pm else ""
        for m in re.finditer(r'\b(?:public\s+|protected\s+|private\s+|static\s+|final\s+|abstract\s+)*class\s+(\w+)', src):
            name=m.group(1)
            ob=src.find('{',m.end())
            if ob==-1: continue
            e=bal(src,ob,'{','}')
            if e==-1: continue
            body=src[ob+1:e]
            while True:
                im=re.search(r'\b(?:public\s+|protected\s+|private\s+|static\s+|final\s+)*class\s+\w+',body)
                if not im: break
                ib=body.find('{',im.end())
                if ib==-1: break
                ie=bal(body,ib,'{','}')
                if ie==-1: break
                body=body[:im.start()]+body[ie+1:]
            marked=[]; allf=[]
            for fm in re.finditer(r'((?:@\w+(?:\([^)]*\))?\s*)*)private\s+(?!static)(?:final\s+)?[\w.$]+(?:\s*<[^;=]*>)?(?:\[\])?\s+(\w+)\s*[;=]', body):
                anns=fm.group(1); fname=fm.group(2)
                allf.append(fname)
                if "@LegacyNonNull" in anns: marked.append(fname)
            if allf:
                new_marked.setdefault(pkg+"."+name,[]).append(marked)
                new_all.setdefault(pkg+"."+name,[]).append(allf)
                if marked: srcfile.setdefault(pkg+"."+name,set()).add(os.path.relpath(p,NEW))

SCOPE=("db.dto","db.vo","common.response","common.exception","apis.internal.controller")
def inscope(k): return any(k.startswith(x+".") for x in SCOPE)
over=[];under=[];unmarked=[]
for key,lists in new_marked.items():
    if not inscope(key): continue
    marked=set().union(*[set(x) for x in lists])
    if key not in old_nonnull:
        if marked: over.append((key,sorted(marked),"구에 대응 클래스 없음"))
        continue
    # 같은 패키지에 같은 단순명이 여러 파일에 있을 수 있다(예: db.dto.ml.CookieDict 3벌).
    # 그 경우 어느 한 벌의 non-null 집합에만 들어 있어도 초과가 아니다.
    kt=set().union(*[set(x[0]) for x in old_nonnull[key]])
    present=set().union(*[set(x) for x in new_all[key]])
    if not marked:
        if kt & present: unmarked.append((key,sorted(kt&present)))
        continue
    extra=marked-kt
    missing=(kt&present)-marked
    if extra: over.append((key,sorted(extra),"구 non-null 아님"))
    if missing: under.append((key,sorted(missing)))

print(f"구 '기본값 없는 non-null' 보유 클래스 {len(old_nonnull)}개")
print(f"신 @LegacyNonNull 부착 클래스 {sum(1 for k,v in new_marked.items() if any(v))}개")
print(f"초과 부착 {len(over)}건 / 누락 {len(under)}건")
for k,v,why in over: print(f"  [초과] {k}: {v}  ({why})  파일={sorted(srcfile.get(k,[]))}")
for k,v in under: print(f"  [누락] {k}: {v}")
print(f"\n미부착(직렬화 전용으로 판정된 것) {len(unmarked)}클래스  [범위={SCOPE}]")
for k,v in sorted(unmarked)[:200]: print(f"  {k}: {len(v)}필드")
