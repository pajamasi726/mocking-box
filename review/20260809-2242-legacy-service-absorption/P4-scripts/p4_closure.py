#!/usr/bin/env python3
"""P4 컨트롤러들의 legalcare.medilawyer.* 임포트 전이 폐포를 구한다."""
import os,re,sys,json
from collections import deque, defaultdict

OLD="/Users/steve/steve/legal-care/medilawyer-boot"
# FQCN -> file
fq2file={}
file2src={}
for root,dirs,files in os.walk(OLD):
    if "/build/" in root or "/.git" in root: continue
    for f in files:
        if not f.endswith(".kt"): continue
        p=os.path.join(root,f)
        src=open(p,encoding="utf-8",errors="replace").read()
        file2src[p]=src
        m=re.search(r'^package\s+([\w.]+)', src, re.M)
        pkg=m.group(1) if m else ""
        for tm in re.finditer(r'^(?:@\w+(?:\([^\n]*\))?\s*)*\s*(?:public |internal |private |open |abstract |sealed |data |enum |annotation |value )*(?:class|interface|object|enum class)\s+(\w+)', src, re.M):
            fq2file.setdefault(pkg+"."+tm.group(1), p)
        # top-level fun/val in file  -> FileKt
        fq2file.setdefault(pkg+"."+os.path.basename(p)[:-3]+"Kt", p)

roots=[
 "module-core/core-api/src/main/kotlin/legalcare/medilawyer/apis/reviewBoost/BoostController.kt",
 "module-core/core-api/src/main/kotlin/legalcare/medilawyer/admin/controller/AdminController.kt",
 "module-core/core-api/src/main/kotlin/legalcare/medilawyer/admin/controller/AdminSeoController.kt",
 "module-core/core-api/src/main/kotlin/legalcare/medilawyer/airtable/controller/AirtableController.kt",
 "module-client/core-client/src/main/kotlin/legalcare/medilawyer/client/ml/controller/MlController.kt",
 "module-core/core-api/src/main/kotlin/legalcare/medilawyer/kafka/controller/KafkaController.kt",
 "module-core/core-api/src/main/kotlin/legalcare/medilawyer/eventScheduler/controller/EventSchedulerController.kt",
 "module-core/core-api/src/main/kotlin/legalcare/medilawyer/schedule/controller/ScheduledController.kt",
 "module-core/core-api/src/main/kotlin/legalcare/medilawyer/apis/internal/controller/InternalAdminWebController.kt",
 "module-core/core-api/src/main/kotlin/legalcare/medilawyer/apis/internal/controller/InternalCrawlingReviewBoostController.kt",
 "module-core/core-api/src/main/kotlin/legalcare/medilawyer/apis/internal/controller/InternalProductReviewBoostController.kt",
 "module-core/core-api/src/main/kotlin/legalcare/medilawyer/apis/internal/controller/InternalReviewedHospitalController.kt",
 "module-core/core-api/src/main/kotlin/legalcare/medilawyer/kafka/KafkaConsumer.kt",
]
roots=[os.path.join(OLD,r) for r in roots]
for r in roots:
    if not os.path.exists(r): print("MISSING ROOT", r, file=sys.stderr)

seen=set(); q=deque(); parent={}
for r in roots:
    if os.path.exists(r): seen.add(r); q.append(r); parent[r]="ROOT"
edges=defaultdict(set)
while q:
    p=q.popleft()
    src=file2src[p]
    imps=set()
    for m in re.finditer(r'^import\s+(legalcare\.medilawyer\.[\w.]+(?:\*)?)(?:\s+as\s+\w+)?\s*$', src, re.M):
        imps.add(m.group(1))
    # same-package references: 같은 패키지 클래스도 폐포에 넣는다
    pm=re.search(r'^package\s+([\w.]+)', src, re.M)
    pkg=pm.group(1) if pm else ""
    for fq,fp in fq2file.items():
        if fq.rsplit(".",1)[0]==pkg and fp!=p:
            simple=fq.rsplit(".",1)[1]
            if re.search(r'\b'+re.escape(simple)+r'\b', src):
                imps.add(fq)
    for imp in imps:
        if imp.endswith(".*"):
            base=imp[:-2]
            # 와일드카드는 과대추정이 된다 — 단순명이 소스에 실제로 등장하는 것만 취한다
            targets=[fp for fq,fp in fq2file.items()
                     if fq.rsplit(".",1)[0]==base
                     and re.search(r'\b'+re.escape(fq.rsplit(".",1)[1])+r'\b', src)]
        else:
            targets=[fq2file[imp]] if imp in fq2file else []
        for t in targets:
            edges[p].add(t)
            if t not in seen:
                seen.add(t); parent[t]=p; q.append(t)

files=sorted(seen)
tot=sum(len(file2src[f].split("\n")) for f in files)
print(f"폐포 파일 {len(files)}개 · 총 {tot}줄")
by=defaultdict(list)
for f in files:
    rel=os.path.relpath(f,OLD)
    seg=rel.split("/main/kotlin/legalcare/medilawyer/")[-1]
    by[seg.split("/")[0]].append((rel,len(file2src[f].split("\n"))))
for k in sorted(by, key=lambda k:-sum(x[1] for x in by[k])):
    print(f"  {sum(x[1] for x in by[k]):6d}줄  {len(by[k]):3d}파일  {k}")
json.dump([os.path.relpath(f,OLD) for f in files], open("/Users/steve/steve/mocking-box/review/20260809-2242-legacy-service-absorption/P4-scripts/p4_closure.json","w"), ensure_ascii=False, indent=1)
