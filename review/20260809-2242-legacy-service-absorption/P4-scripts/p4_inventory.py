#!/usr/bin/env python3
"""130경로 최종 인벤토리 + 빠짐0·유령0 양방향 대조."""
import json, os, sys, subprocess, collections

BASE="/Users/steve/steve/mocking-box/review/20260809-2242-legacy-service-absorption/P4-scripts"
old=json.load(open(f"{BASE}/old_endpoints.json"))
new=json.load(open(f"{BASE}/new_endpoints.json"))

# 구: *Controller.kt 129 + HealthCheck 1 = 130
HEALTH={"file":"module-core/core-common/src/main/kotlin/legalcare/medilawyer/common/health/HealthCheck.kt",
        "line":0,"class":"HealthCheck","method":"GET","path":"/api/v1/health","ann":"GetMapping"}
hc=subprocess.run(["grep","-n","Mapping","/Users/steve/steve/legal-care/medilawyer-boot/module-core/core-common/src/main/kotlin/legalcare/medilawyer/common/health/HealthCheck.kt"],
                  capture_output=True,text=True).stdout.strip()
old_all=old+[HEALTH]

PHASE={
 "PublicHospitalController":"P1","CommunityController":"P1","PostWordController":"P1","StoreChannelController":"P1",
 "PostController":"P1","PostReplyController":"P1","PostReplyAiRecommendController":"P1",
 "StoreController":"P2","AppUserController":"P2","AppUserStoreController":"P2","AppUserChannelAccountController":"P2",
 "TeamController":"P2","TeamAppUserController":"P2","TeamAuthController":"P2","TeamChannelAccountController":"P2",
 "OrganizationController":"P2","OrganizationAppUserController":"P2","NoticeController":"P2","ProductController":"P2",
 "AuthController":"P3","OtpController":"P3","JwtController":"P3",
 "BoostController":"P4","AdminController":"P4","AdminSeoController":"P4","AirtableController":"P4",
 "MlController":"P4","KafkaController":"P4","EventSchedulerController":"P4","ScheduledController":"P4",
 "InternalAdminWebController":"P4","InternalCrawlingReviewBoostController":"P4",
 "InternalProductReviewBoostController":"P4","InternalReviewedHospitalController":"P4",
 "HealthCheck":"P0","ReviewBoostV2Controller":"선행(리뷰부스터 v2 이관)",
}

newidx=collections.defaultdict(list)
for r in new: newidx[(r["method"],r["path"])].append(r)

rows=[]; miss=[]
for r in old_all:
    key=(r["method"],r["path"])
    hit=newidx.get(key,[])
    rows.append({**r,"phase":PHASE.get(r["class"],"?"),
                 "new":"; ".join(f'{h["file"]}:{h["line"]}' for h in hit) if hit else "",
                 "status":"이식완료" if hit else "미이식"})
    if not hit: miss.append(r)

oldkeys=set((r["method"],r["path"]) for r in old_all)
ghost=[r for r in new if (r["method"],r["path"]) not in oldkeys
       and ("/modules/legacy/" in r["file"])]

print(f"구 엔드포인트 총 {len(old_all)}  (컨트롤러 {len(old)} + HealthCheck 1)")
print(f"이식완료 {len(rows)-len(miss)} / 미이식 {len(miss)}")
print(f"유령(신 modules/legacy 에만 있는 매핑) {len(ghost)}")
by=collections.Counter(r["phase"] for r in rows if r["status"]=="미이식")
for k,v in sorted(by.items()): print(f"   미이식 {v:3d}  {k}")
if ghost:
    for g in ghost: print("   유령:", g["method"], g["path"], g["file"]+":"+str(g["line"]))

hdr=["단계","HTTP","경로","구 위치(파일:라인)","신 위치(파일:라인)","판정"]
out=["\t".join(hdr)]
for r in sorted(rows,key=lambda r:(r["phase"],r["class"],r["line"])):
    out.append("\t".join([r["phase"],r["method"],r["path"],
                          f'{r["file"]}:{r["line"]}',r["new"],r["status"]]))
open(f"{BASE}/../P4-inventory-130.tsv","w").write("\n".join(out)+"\n")
print("→ P4-inventory-130.tsv", len(out)-1, "행")
