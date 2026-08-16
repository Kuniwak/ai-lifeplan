# Where the environment tables come from

> **地の文は未回復である。**個人情報の混入を断つため、いったん章構成と
> 出典の引用だけを残して地の文を削除した。復元は別のセッションで行う。
> **書き戻すときは、世帯の帳票ではなく公開資料だけを根拠にすること。**


> **金額を伏せてある。**この文書が引く感度や突合の絶対額は公開していない。
> 割合と向きは残してあり、主張はそれで通る。**`data/` にいるのは架空世帯なので、
> どのみちこれらの絶対額は再現しない。**

> **English note.** This directory holds what a household does *not* choose —
> inflation, real wage growth, investment return, pension levels, the cost
> growth of medical and nursing care. The figures come from the government's
> 財政検証 and from OECD projections.
>
> **The body is in Japanese on purpose**: it quotes those sources, and a
> quotation stays in its own language.
>
> `scenario/` holds the alternative economies the comparison and the sweep use.

# 環境の表の出どころ

## household.tsv の子は 2 人である

> 結婚持続期間 15〜19 年の夫婦の完結出生子ども数は、2002 年（第 12 回）調査までは
> 2.2 人前後で安定的に推移していたが、その後低下し、今回調査では **1.90 人** となり
> 最低値を更新した。
>
> SOURCE: 国立社会保障・人口問題研究所「第16回出生動向調査（結婚と出産に関する
> 全国調査）結果の概要」6.1 完結出生子ども数
> https://www.ipss.go.jp/ps-doukou/j/doukou16/JNFS16gaiyo.pdf

1.90 人を四捨五入して 2 人にしてある。**この選び方の代償として、児童手当の
「第 3 子以降」（月 3 万円）をこの計画の誰も踏まなくなった**——規定は
`data/law/national/child-allowance-limits.tsv` にあり `law` のテストが見ているが、
計画としての検算は失われている。

## residence.tsv は八王子市である

世帯の額を統計に合わせた結果、世田谷区の住居費では初年度から破綻する。
理由と数字は [../controllable/README.md](../controllable/README.md)
「住まいは八王子市の 70 ㎡ で揃えてある」にある。

## inflation-target.tsv

### 一般物価とは別の指数で動く費目がある

#### 学費と子生活費が 100% であることの感度

#### どれだけ効くか

#### 生活費は持家の帰属家賃を買っていない

> 　　　　　　　　　　　　　　ウエイト　　2025年12月 指数
> 総　　　　　合　　　　　　　　10000　　　113.0
> 持家の帰属家賃を除く総合　　　 8420　　　115.3

#### 医療費の実績は 100% より 0% に近い

##### 令和8年度の改定（`source/r8-shinryou-housyuu-kaiteiritsu.md`）

> １．診療報酬
> 　＋3.09％（令和８年度及び令和９年度の２年度平均。…）
> ※２ うち、物価対応分 ＋0.76％（令和８年度及び令和９年度の２年度平均。…）
> ２．薬価等
> 　合計　　▲0.87％（国費▲1,063 億円程度）

> 　実際の経済・物価の動向が令和８年度診療報酬改定時の見通しから大きく変動し、
> 　医療機関等の経営状況に支障が生じた場合には、…令和９年度予算編成において
> 　加減算を含め更なる必要な調整を行う。

#### 住宅維持費の向きは一方向だが、比は窓で 1.11〜2.55 に振れる

##### 1 つを選ばず、幅で示す

##### 比ではなく差で持つ
