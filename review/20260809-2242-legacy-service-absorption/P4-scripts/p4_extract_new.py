#!/usr/bin/env python3
"""신 booster Java 컨트롤러에서 @*Mapping 엔드포인트를 괄호균형으로 추출.
modules/legacy 전체 + 다른 모듈 중 /api/v1·/api/v2·/internal 을 문자 그대로 서빙하는 것."""
import os, re, json, sys

ROOTS = [
    "/Users/steve/steve/legalcare-renew-prodsync-wt/apps/booster-app/modules",
    "/Users/steve/steve/legalcare-renew-prodsync-wt/apps/booster-app/app/src/main",
]
ANN = ["RequestMapping","GetMapping","PostMapping","PutMapping","PatchMapping","DeleteMapping"]
ANN_METHOD = {"GetMapping":"GET","PostMapping":"POST","PutMapping":"PUT","PatchMapping":"PATCH","DeleteMapping":"DELETE"}

def strip_comments(src):
    out=[]; i=0; n=len(src)
    while i<n:
        c=src[i]
        if c=='"':
            j=i+1
            while j<n:
                if src[j]=='\\': j+=2; continue
                if src[j]=='"': j+=1; break
                j+=1
            out.append(src[i:j]); i=j; continue
        if c=="'":
            j=i+1
            while j<n:
                if src[j]=='\\': j+=2; continue
                if src[j]=="'": j+=1; break
                j+=1
            out.append(src[i:j]); i=j; continue
        if src[i:i+2]=='//':
            j=src.find('\n',i); j=n if j==-1 else j
            out.append(' '*(j-i)); i=j; continue
        if src[i:i+2]=='/*':
            j=src.find('*/',i+2); j=n if j==-1 else j+2
            out.append(' '*(j-i)); i=j; continue
        out.append(c); i+=1
    return "".join(out)

def balanced(src,start):
    depth=0;i=start;n=len(src)
    while i<n:
        c=src[i]
        if c=='"':
            i+=1
            while i<n:
                if src[i]=='\\': i+=2; continue
                if src[i]=='"': i+=1; break
                i+=1
            continue
        if c=='(': depth+=1
        elif c==')':
            depth-=1
            if depth==0: return i
        i+=1
    return -1

def paths_from(args):
    if args is None: return [""]
    kept=[]
    for sm in re.finditer(r'"((?:[^"\\]|\\.)*)"', args):
        before=args[:sm.start()]
        km=None
        for kw in ("produces","consumes","headers","params","name"):
            for k in re.finditer(r'\b'+kw+r'\s*=', before):
                if km is None or k.start()>km[1]: km=(kw,k.start())
        vm=None
        for kw in ("value","path"):
            for k in re.finditer(r'\b'+kw+r'\s*=', before):
                if vm is None or k.start()>vm[1]: vm=(kw,k.start())
        if km and (vm is None or km[1]>vm[1]): continue
        kept.append(sm.group(1))
    return kept if kept else [""]

def methods_from(name,args):
    if name in ANN_METHOD: return [ANN_METHOD[name]]
    if args:
        ms=re.findall(r'RequestMethod\.([A-Z]+)',args)
        if ms: return ms
    return ["ANY"]

def join(base,sub):
    if not base and not sub: return "/"
    b=base.rstrip("/"); s=sub
    if s and not s.startswith("/"): s="/"+s
    return (b+s) or "/"

res=[]
for root in ROOTS:
    for dirpath,dirs,files in os.walk(root):
        if "/build/" in dirpath or "/src/test/" in dirpath: continue
        for f in files:
            if not f.endswith(".java"): continue
            p=os.path.join(dirpath,f)
            raw=open(p,encoding="utf-8",errors="replace").read()
            if "@RestController" not in raw and "@Controller" not in raw: continue
            src=strip_comments(raw)
            def lineno(i): return src[:i].count("\n")+1
            cls=re.search(r'^\s*(?:public\s+|final\s+|abstract\s+)*class\s+(\w+)',src,re.M)
            cls_idx=cls.start() if cls else len(src)
            occ=[]
            for m in re.finditer(r'@('+"|".join(ANN)+r')\b',src):
                j=m.end()
                while j<len(src) and src[j] in ' \t\r\n': j+=1
                if j<len(src) and src[j]=='(':
                    e=balanced(src,j); occ.append((m.group(1),src[j+1:e],m.start()))
                else: occ.append((m.group(1),None,m.start()))
            base=""
            for name,args,at in occ:
                if at<cls_idx and name=="RequestMapping":
                    pp=paths_from(args); base=pp[0] if pp else ""
            for name,args,at in occ:
                if at<cls_idx: continue
                for pth in paths_from(args):
                    for meth in methods_from(name,args):
                        res.append({"file":os.path.relpath(p,"/Users/steve/steve/legalcare-renew-prodsync-wt"),
                                    "line":lineno(at),"class":cls.group(1) if cls else "?",
                                    "method":meth,"path":join(base,pth)})
res.sort(key=lambda r:(r["file"],r["line"]))
json.dump(res,open("/Users/steve/steve/mocking-box/review/20260809-2242-legacy-service-absorption/P4-scripts/new_endpoints.json","w"),ensure_ascii=False,indent=1)
legacy=[r for r in res if "/modules/legacy/" in r["file"]]
print("신 전체 매핑", len(res), " / modules/legacy", len(legacy))
import collections
for k,v in sorted(collections.Counter(r["class"] for r in legacy).items()): print(f"  {v:4d}  {k}")
