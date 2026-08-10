#!/usr/bin/env python3
"""구 Kotlin 컨트롤러 전수에서 @*Mapping 엔드포인트를 괄호균형으로 추출."""
import os, re, sys, json

OLD = "/Users/steve/steve/legal-care/medilawyer-boot"
MAP_ANN = ["RequestMapping","GetMapping","PostMapping","PutMapping","PatchMapping","DeleteMapping"]
ANN_METHOD = {"GetMapping":"GET","PostMapping":"POST","PutMapping":"PUT","PatchMapping":"PATCH","DeleteMapping":"DELETE"}

def balanced_span(src, start):
    """start = '(' 의 인덱스. 문자열/문자 리터럴 무시하고 짝 맞는 ')' 인덱스 반환."""
    depth=0; i=start; n=len(src)
    while i<n:
        c=src[i]
        if c=='"':
            # triple quote?
            if src[i:i+3]=='"""':
                j=src.find('"""', i+3)
                i = (j+3) if j!=-1 else n
                continue
            i+=1
            while i<n:
                if src[i]=='\\': i+=2; continue
                if src[i]=='"': i+=1; break
                i+=1
            continue
        if c=="'":
            i+=1
            while i<n:
                if src[i]=='\\': i+=2; continue
                if src[i]=="'": i+=1; break
                i+=1
            continue
        if c=='(': depth+=1
        elif c==')':
            depth-=1
            if depth==0: return i
        i+=1
    return -1

def ann_occurrences(src):
    """(annName, argstr|None, atIndex) 목록. 다중행 인자 포함."""
    out=[]
    for m in re.finditer(r'@(' + "|".join(MAP_ANN) + r')\b', src):
        name=m.group(1); at=m.start()
        j=m.end()
        while j<len(src) and src[j] in ' \t\r\n': j+=1
        if j<len(src) and src[j]=='(':
            e=balanced_span(src,j)
            out.append((name, src[j+1:e], at))
        else:
            out.append((name, None, at))
    return out

def paths_from_args(args):
    if args is None: return [""]
    # value = [...] 또는 첫 위치인자
    ps=[]
    for sm in re.finditer(r'"((?:[^"\\]|\\.)*)"', args):
        # method = [RequestMethod.X] 는 문자열 없음. produces/consumes 는 제외해야 함
        ps.append(sm.group(1))
    if not ps: return [""]
    # produces/consumes/headers/params 안의 문자열 제거
    kept=[]
    for sm in re.finditer(r'"((?:[^"\\]|\\.)*)"', args):
        before=args[:sm.start()]
        # 가장 가까운 키워드 판정
        km=None
        for kw in ("produces","consumes","headers","params","name"):
            for k in re.finditer(r'\b'+kw+r'\s*=', before):
                if km is None or k.start()>km[1]: km=(kw,k.start())
        # 그 키워드 이후에 다른 키워드(value/path)가 있으면 무효화
        vm=None
        for kw in ("value","path"):
            for k in re.finditer(r'\b'+kw+r'\s*=', before):
                if vm is None or k.start()>vm[1]: vm=(kw,k.start())
        if km and (vm is None or km[1]>vm[1]):
            continue
        kept.append(sm.group(1))
    return kept if kept else [""]

def methods_from_args(name,args):
    if name in ANN_METHOD: return [ANN_METHOD[name]]
    if args:
        ms=re.findall(r'RequestMethod\.([A-Z]+)', args)
        if ms: return ms
    return ["ANY"]

def join(base, sub):
    if not base and not sub: return "/"
    b=base.rstrip("/"); s=sub
    if s and not s.startswith("/"): s="/"+s
    r=(b+s) or "/"
    return r

results=[]
for root,dirs,files in os.walk(OLD):
    if "/build/" in root or "/.git" in root: continue
    for f in files:
        if not f.endswith("Controller.kt"): continue
        p=os.path.join(root,f)
        src=open(p,encoding="utf-8").read()
        lines=src.split("\n")
        def lineno(idx): return src[:idx].count("\n")+1
        occ=ann_occurrences(src)
        # 클래스 레벨 = 'class ' 선언 앞에 오는 @RequestMapping
        cls_m=re.search(r'^\s*(?:open\s+|abstract\s+|final\s+)*class\s+(\w+)', src, re.M)
        cls_idx=cls_m.start() if cls_m else len(src)
        base=""
        base_occ=[o for o in occ if o[2]<cls_idx]
        for name,args,at in base_occ:
            if name=="RequestMapping":
                pp=paths_from_args(args)
                base=pp[0] if pp else ""
        for name,args,at in occ:
            if at<cls_idx: continue
            for pth in paths_from_args(args):
                for meth in methods_from_args(name,args):
                    results.append({
                        "file": os.path.relpath(p,OLD),
                        "line": lineno(at),
                        "class": cls_m.group(1) if cls_m else "?",
                        "method": meth,
                        "path": join(base,pth),
                        "ann": name,
                    })
results.sort(key=lambda r:(r["file"],r["line"]))
json.dump(results, open("/Users/steve/steve/mocking-box/review/20260809-2242-legacy-service-absorption/P4-scripts/old_endpoints.json","w"), ensure_ascii=False, indent=1)
print("총", len(results))
from collections import Counter
c=Counter(r["class"] for r in results)
for k,v in sorted(c.items()): print(f"{v:4d}  {k}")
