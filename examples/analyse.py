# -*- coding: utf-8 -*-
import io,csv,statistics,json,collections,math
rows=list(csv.DictReader(io.open('out/sweep/cells.tsv',encoding='utf-8'),delimiter='\t'))
AX=['経済','生活費','年金受給開始','住まい','金融危機']
ENV={'経済','金融危機'}
OKU=1e8
for r in rows:
    r['net']=int(r['純資産2090'])/OKU; r['ruin']= r['破産年']!=''
vals=[r['net'] for r in rows]; n=len(rows)
D={'n':n}; q=statistics.quantiles(vals,n=4)
D['dist']={'min':min(vals),'q1':q[0],'med':statistics.median(vals),'q3':q[2],'max':max(vals),
           'ruin':sum(r['ruin'] for r in rows)/n,'ruinN':sum(r['ruin'] for r in rows)}
step=0.5; x=math.floor(min(vals)/step)*step; hist=[]
while x<max(vals)+step:
    sel=[r for r in rows if x<=r['net']<x+step]
    if sel: hist.append({'lo':round(x,2),'n':len(sel),'ruin':sum(r['ruin'] for r in sel)})
    x+=step
D['hist']=hist
gm=sum(vals)/n; sst=sum((v-gm)**2 for v in vals); eta=[]
for a in AX:
    g=collections.defaultdict(list)
    for r in rows: g[r[a]].append(r['net'])
    eta.append({'name':a,'v':sum(len(v)*((sum(v)/len(v))-gm)**2 for v in g.values())/sst,'env':a in ENV})
eta.sort(key=lambda e:-e['v']); D['eta']=eta; D['etaMainSum']=sum(e['v'] for e in eta)
econ=[]
for name in sorted({r['経済'] for r in rows}):
    sel=[r for r in rows if r['経済']==name]; v=sorted(r['net'] for r in sel); qq=statistics.quantiles(v,n=4)
    econ.append({'name':name,'min':v[0],'q1':qq[0],'med':statistics.median(v),'q3':qq[2],'max':v[-1],
                 'ruin':sum(r['ruin'] for r in sel)/len(sel),'n':len(sel)})
econ.sort(key=lambda e:-e['med']); D['econ']=econ
D['dials']={a:[{'lv':lv,'ruin':sum(r['ruin'] for r in sel)/len(sel),
                'med':statistics.median([r['net'] for r in sel]),'env':a in ENV}
               for lv in sorted({r[a] for r in rows}) for sel in [[r for r in rows if r[a]==lv]]] for a in AX}
ages=[int(r['破産年']) for r in rows if r['ruin']]
if ages:
    D['ruinAge']={'min':min(ages),'med':int(statistics.median(ages)),'max':max(ages)}
    bins=collections.Counter((a//5)*5 for a in ages)
    D['ruinAgeBins']=[{'year':y,'n':bins[y]} for y in sorted(bins)]
lr=[r for r in rows if r['最後の手段']]
D['resort']={'used':len(lr),
 'byMeasure':[{'name':m,'n':sum(1 for r in lr if r['最後の手段']==m)} for m in sorted({r['最後の手段'] for r in lr})],
 'byEconomy':[{'name':e,'n':sum(1 for r in rows if r['経済']==e),'used':sum(1 for r in lr if r['経済']==e)} for e in sorted({r['経済'] for r in rows})]}
if lr:
    ry=[int(r['最後の手段の年']) for r in lr if r['最後の手段の年']]
    if ry: D['resort']['years']={'min':min(ry),'med':int(statistics.median(ry)),'max':max(ry)}
heat=[]
for w in sorted({r['住まい'] for r in rows}):
    for c in sorted({r['生活費'] for r in rows}):
        sel=[r for r in rows if r['住まい']==w and r['生活費']==c]
        heat.append({'住まい':w,'生活費':c,'n':len(sel),'ruin':sum(r['ruin'] for r in sel)/len(sel),
                     'med':statistics.median([r['net'] for r in sel])})
D['heat']=heat
io.open('out/sweep/data.json','w').write(json.dumps(D,ensure_ascii=False))
print('n=%d ruin=%.1f%% (%d) med=%+.2f min=%+.2f max=%+.2f'%(n,100*D['dist']['ruin'],D['dist']['ruinN'],D['dist']['med'],D['dist']['min'],D['dist']['max']))
print('eta:',[(e['name'],round(e['v']*100,1)) for e in eta])
print('econ:',[(e['name'],round(e['med'],2),round(100*e['ruin'],1)) for e in econ])
print('resort:',D['resort']['used'],D['resort'].get('years'),'ruinAge:',D.get('ruinAge'))
