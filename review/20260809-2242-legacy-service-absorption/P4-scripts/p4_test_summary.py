#!/usr/bin/env python3
"""booster 전체 테스트 결과 집계(모듈별 + 총계). ./gradlew test 후 실행."""
import glob, re, os, collections
BASE="/Users/steve/steve/legalcare-renew-prodsync-wt/apps/booster-app"
tot=collections.Counter(); per=collections.defaultdict(collections.Counter); classes=0
fails=[]
for p in glob.glob(f"{BASE}/*/build/test-results/test/TEST-*.xml") + \
         glob.glob(f"{BASE}/modules/*/build/test-results/test/TEST-*.xml"):
    mod=p.replace(BASE+"/","").split("/build/")[0]
    h=open(p,encoding="utf-8",errors="replace").read()
    head=h[:3000]
    def g(k):
        m=re.search(k+r'="(\d+)"',head); return int(m.group(1)) if m else 0
    classes+=1
    for k in ("tests","failures","errors","skipped"):
        tot[k]+=g(k); per[mod][k]+=g(k)
    if g("failures")+g("errors"):
        for m in re.finditer(r'<testcase name="([^"]+)" classname="([^"]+)"[^/>]*>\s*<(failure|error)', h):
            fails.append(f"{m.group(2)}.{m.group(1)}")
print(f"클래스 {classes}  tests={tot['tests']} failures={tot['failures']} errors={tot['errors']} skipped={tot['skipped']}")
for m in sorted(per): print(f"  {m:22s} {per[m]['tests']:4d}")
if fails:
    print("\n실패:")
    for f in fails: print("  ", f)
