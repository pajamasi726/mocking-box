#!/usr/bin/env python3
"""구 Kotlin ↔ 신 Java: 클래스 단위 필드명·선언순서 + @JsonPropertyOrder 일치 기계 대조.
이름이 같은 클래스만 짝짓는다(구에 있는데 신에 없으면 '미이식', 그 반대면 '유령')."""
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
        if c=="'":
            i+=1
            while i<n:
                if src[i]=='\\': i+=2; continue
                if src[i]=="'": i+=1; break
                i+=1
            continue
        if c==op: d+=1
        elif c==cl:
            d-=1
            if d==0: return i
        i+=1
    return -1

def split_top(body):
    parts=[];depth=0;cur=""
    i=0
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

def kt_scan(root):
    out={}
    for dp,dn,fn in os.walk(root):
        if "/build/" in dp: continue
        for f in fn:
            if not f.endswith(".kt"): continue
            p=os.path.join(dp,f)
            src=open(p,encoding="utf-8",errors="replace").read()
            src=re.sub(r'/\*.*?\*/','',src,flags=re.S)
            src=re.sub(r'//[^\n]*','',src)
            pkgm=re.search(r'^package\s+([\w.]+)', src, re.M)
            pkg=pkgm.group(1).replace("legalcare.medilawyer.","") if pkgm else ""
            for m in re.finditer(r'\b(?:data\s+)?class\s+(\w+)\s*(?:<[^>]*>\s*)?\(', src):
                name=m.group(1); op=src.index('(',m.end()-1); e=bal(src,op)
                if e==-1: continue
                names=[]
                for fpart in split_top(src[op+1:e]):
                    fm=re.search(r'\b(?:val|var)\s+(\w+)\s*:', fpart)
                    if fm: names.append(fm.group(1))
                if names: out.setdefault(pkg+"."+name,[]).append((names,os.path.relpath(p,OLD)))
    return out

FIELD=re.compile(r'(?:^|\n)\s*private\s+(?!static)(?:final\s+)?[\w.$]+(?:\s*<[^;=]*>)?(?:\[\])?\s+(\w+)\s*[;=]')
def java_scan(root):
    out={}; pins={}
    for dp,dn,fn in os.walk(root):
        if "/build/" in dp: continue
        for f in fn:
            if not f.endswith(".java"): continue
            p=os.path.join(dp,f)
            raw=open(p,encoding="utf-8",errors="replace").read()
            src=re.sub(r'/\*.*?\*/','',raw,flags=re.S)
            src=re.sub(r'//[^\n]*','',src)
            pkgm=re.search(r'^package\s+([\w.]+)', src, re.M)
            pkg=pkgm.group(1).replace("legalcare.medilawyer.legacy.","") if pkgm else ""
            for m in re.finditer(r'\b(?:public\s+|protected\s+|private\s+|static\s+|final\s+|abstract\s+)*class\s+(\w+)', src):
                name=m.group(1)
                ob=src.find('{',m.end())
                if ob==-1: continue
                e=bal(src,ob,'{','}')
                if e==-1: continue
                body=src[ob+1:e]
                # 중첩 클래스 본문 제거(자기 필드만 남긴다)
                while True:
                    im=re.search(r'\b(?:public\s+|protected\s+|private\s+|static\s+|final\s+)*class\s+\w+',body)
                    if not im: break
                    ib=body.find('{',im.end())
                    if ib==-1: break
                    ie=bal(body,ib,'{','}')
                    if ie==-1: break
                    body=body[:im.start()]+body[ie+1:]
                names=FIELD.findall(body)
                if names: out.setdefault(pkg+"."+name,[]).append((names,os.path.relpath(p,NEW)))
                # 이 클래스 선언 직전의 @JsonPropertyOrder
                head=src[:m.start()]
                jm=None
                for x in re.finditer(r'@JsonPropertyOrder\s*\(',head): jm=x
                if jm and not re.search(r'class\s+\w+', head[jm.end():]):
                    je=bal(src,src.index('(',jm.end()-1))
                    pins[(os.path.relpath(p,NEW), pkg+"."+name)]=re.findall(r'"([^"]*)"', src[jm.end():je])
    return out,pins

kt=kt_scan(OLD)
jv,pins=java_scan(NEW)
SCOPE=("db.dto","db.vo","common.response","common.exception",
       "apis.internal.controller.request","apis.internal.controller.response")
def inscope(k): return any(k.startswith(s+".") for s in SCOPE)
common=sorted(k for k in (set(kt)&set(jv)) if inscope(k))
only_old=sorted(k for k in (set(kt)-set(jv)) if inscope(k))
only_new=sorted(k for k in (set(jv)-set(kt)) if inscope(k))
bad=[]
for n in common:
    kolist=kt[n]; jolist=jv[n]
    matched=False; detail=None
    for ko,kp in kolist:
        for jo,jp in jolist:
            if ko==jo:
                matched=True
                pin=pins.get((jp,n))
                if pin is not None and pin!=jo:
                    bad.append((n,"@JsonPropertyOrder 불일치",pin,jo,kp,jp))
                break
            detail=(ko,jo,kp,jp)
        if matched: break
    if not matched and detail:
        bad.append((n,"필드명·순서",detail[0],detail[1],detail[2],detail[3]))
print(f"대조 범위 {SCOPE}")
print(f"짝지어진 클래스 {len(common)}쌍")
print(f"구에만 있음(미이식 후보) {len(only_old)}: {only_old}")
print(f"신에만 있음(유령 후보) {len(only_new)}: {only_new}")
print(f"불일치 {len(bad)}건")
for b in bad[:60]:
    print(f"  [{b[1]}] {b[0]}")
    print(f"      구 {b[4]}: {b[2]}")
    print(f"      신 {b[5]}: {b[3]}")
