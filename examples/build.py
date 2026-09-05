# -*- coding: utf-8 -*-
"""Render out/sweep/data.json as examples/sweep-report.html (English) and
examples/sweep-report_ja.html (Japanese).

All prose lives in the TEXT table below, one entry per language. The HTML
skeleton is shared. Statements that depend on the numbers (which condition is
largest, whether the worst economy ruins every combination) are computed, so the
prose never contradicts the data it sits next to.
"""
import io, json, sys

D = json.load(io.open('out/sweep/data.json', encoding='utf-8'))
res = D['resort']
ry = res.get('years', {})
W = 520


def oku(v): return ('+' if v >= 0 else '−') + ('%.2f' % abs(v))
def pct(v): return '%.1f%%' % (v * 100)
def com(v): return '{:,}'.format(v)
def man(v): return '{:,}'.format(int(round(v / 10000)))
def yrs(v): return '%.1f' % v
def bar(v, mx, w=W): return max(2, int(round(v / mx * w))) if mx else 2


# ---------------------------------------------------------------- facts --
top = D['eta'][0]                         # axis with the largest η²
dials = [e for e in D['eta'] if not e['env']]
top_dial = dials[0]                       # largest axis the household chooses
econ_eta = next(e for e in D['eta'] if e['name'] == '経済')
worst_econ = max(D['econ'], key=lambda e: e['ruin'])
house = {r['lv']: r for r in D['dials']['住まい']}
hs = D['housing']
hyears = hs['years']
hlast = hyears[-1]
cover_last = {e['name']: e['rows'][-1]['cover'] for e in hs['econ']}
cover_first = {e['name']: e['rows'][0]['cover'] for e in hs['econ']}
fast = min(cover_last.values())                            # prices rise fastest
econ_fast = [e['name'] for e in hs['econ'] if cover_last[e['name']] == fast]
econ_slow = max(cover_last, key=lambda k: cover_last[k])

# The year the renting scenario moves to a smaller flat: the one year where the
# rent falls in real terms. Read off the series rather than written down here,
# so the prose cannot drift from the table it sits under.
_rows0 = hs['econ'][0]['rows']
MOVE_YEAR = next(cur['year'] for prev, cur in zip(_rows0, _rows0[1:])
                 if cur['rent'] / cur['level'] < prev['rent'] / prev['level'])
resort_econ = [e['name'] for e in res['byEconomy'] if e['used']]
econ_names = [e['name'] for e in D['econ']]
worst_all_fail = worst_econ['ruin'] >= 1.0

# ---------------------------------------------------------------- text ---
TEXT = {}

TEXT['en'] = dict(
    lang='en',
    kicker='lifeplan / every combination of five conditions / example',
    title='%(n)s scenarios this household could follow',
    lede=(
        'For an invented household, every combination of three choices the household makes and two outside conditions it cannot choose '
        'was turned into a scenario, %(n)s in all. In some the household still has assets at 100; in others the assets run out. '
        'The point of computing every combination is to see <strong>which conditions shape the path of the assets, and by how much</strong>.'),

    h_axes='What conditions were assumed',
    p_axes='Computing every combination of the conditions shows which option to take.',
    p_method=(
        'There are two kinds: conditions nobody controls (prices, wage growth, investment returns, the timing of a crash) '
        "and decisions (housing, work, spending, children's schooling, when to draw the pension). "
        'Computing a scenario for each combination shows which decisions matter.'),
    th_axis='Condition', th_chosen='Household chooses?', th_what='What the options are',
    axes=[
        ('経済', 'no', 'seven published economic projections'),
        ('金融危機', 'no', 'no crash, a −20%% crash in one of five chosen years, or a crash in two consecutive years'),
        ('生活費', 'yes', 'the 家計調査 average for the head\'s age band, or that figure ±¥40,000 a month'),
        ('年金受給開始', 'yes', 'as in the plan (70), or 65, 70, or 75 for both'),
        ('住まい', 'yes', 'buy a 70㎡ flat in 2023, or rent for life, moving to a one-bedroom once the children leave'),
    ],
    cap_axes=(
        '7 × 7 × 3 × 4 × 2 = <b>%(n)s</b> combinations. Each added condition multiplies the number of plans to run, '
        'so choosing the conditions is also choosing how long the whole run takes.'),


    h_eta='Which condition moves the result',
    sub_eta='Share of the spread each condition accounts for (variance explained, η²)',
    p_eta=(
        'Roughly, η² is how much a condition matters. '
        'Precisely, it is the spread that disappears when that condition is held at one option.'),
    th_share='Share of the spread',
    cap_eta=(
        'Orange bars are conditions the household cannot choose; blue bars are decisions it can make. '
        'The five conditions acting separately explain <b>%(etaMain)s</b> of the spread. The remainder comes from '
        'conditions acting together, where the effect of one depends on the option chosen for another.'),

    h_econ='The economy is not a choice',
    sub_econ='For each economy, every combination of the other four conditions',
    th_econ='Economy', th_worst='worst', th_median='median', th_best='best', th_ruin='ruin rate',
    cap_econ='Hundred-million yen, today\'s prices. The shaded row is the economy the original plan assumes.',
    callout_econ_all=(
        '<strong>In the harshest economy, every combination fails.</strong> No setting of the household\'s '
        'decisions keeps it solvent there. This is the sweep\'s most useful output: it shows where the problem '
        'lies outside the household\'s choices.'),
    callout_econ_most=(
        '<strong>In the harshest economy (%(worstEcon)s), %(worstRuin)s of combinations fail.</strong> '
        'Only the most favourable settings of the household\'s decisions survive there. This is the sweep\'s '
        'most useful output: it shows how much of the problem is outside what the household can choose.'),

    h_dials='Each decision on its own',
    sub_dials='Median net worth and ruin rate for each option, across every combination of the other conditions',
    th_housing='Housing', th_living='Living cost', th_pension='Pension start', th_crash='Crash', th_ruin_s='ruin rate',
    cap_housing_bigger=(
        'Housing is the largest decision the household controls, and it moves the result more than the economy does.'),
    cap_housing_smaller=(
        'Housing is the largest decision the household controls. It moves the result less than the economy, '
        'but more than every other decision.'),
    cap_living='Spending is a real lever, but a smaller one than housing.',
    cap_pension=(
        'Deferring the pension raises the median, but the years spent waiting have to be paid for by drawing down savings.'),
    cap_crash=(
        'The same percentage fall costs a different amount depending on when it lands and how much is held at the time.'),
    p_dials_note=(
        'Where the rows of a table barely differ, that decision barely matters for this household: %(smallDials)s '
        '%(smallVerb)s for under 1%% of the spread. That is a result too. It says which questions the household can '
        'stop deliberating over, and which one (%(topDial)s) deserves the time.'),

    h_heat='Housing and living cost together',
    sub_heat='Each row covers every combination of the other three conditions',
    cap_heat='The median and ruin rate in each row are taken over those combinations.',

    h_resort='What happens after the money runs out',
    sub_resort='When the assets run out, the household either sells the home and rents, or borrows against it',
    p_resort=(
        'Both require the mortgage to be repaid first. A home already pledged cannot be pledged again, '
        'so selling means clearing the loan out of the proceeds. <strong>Both are evaluated and the better one is taken.</strong>'),
    th_measure='Measure', th_times='times taken',
    cap_resortM=(
        'Taken in <b>%(resortUsed)s</b> cells. The earliest year it was needed was <b>%(resortMin)s</b>, '
        'the median year <b>%(resortMed)s</b>, and the latest <b>%(resortMax)s</b>.'),
    th_cells_used='combinations that needed it', th_share_s='share',
    cap_resortE=(
        'Only these economies ever force the home into cash: %(resortEcons)s. '
        '<b>Whether the home has to be sold is settled by the economy before the household chooses anything.</b>'),


    h_housing='How the rent and the collateral move',
    sub_housing='For each economy, the rent of the renting scenario and the value the home can be pledged for',
    p_housing=(
        'Rent is carried up by prices. The value the home can be pledged for is the land value on the property tax '
        'notice, and it is held at that figure in cash terms for the whole span. The two do not move together, '
        'so the later the home is turned into cash, the fewer years of rent it buys.'),
    cap_rent=(
        '<b>Rent, cash terms.</b> Yen a year, cash of the day. In real terms the rent is flat until the move to a one-bedroom in %(moveYear)s, '
        'so every change here except that step is prices. Seven economies produce only <b>%(paths)s</b> price paths, '
        'and economies that share a path share a row.'),
    cap_real=(
        '<b>Pledged value, real terms.</b> The same ¥%(collateralYen)s, deflated to the prices of the plan\'s first year. '
        'Nothing about the land has changed; the figure falls because the plan holds it in cash terms while '
        'prices rise.'),
    cap_cover=(
        '<b>Years of rent the sale covers.</b> The pledged value is ¥%(collateralYen)s, of which selling nets %(proceedRate)s, or ¥%(proceedsYen)s. '
        'That is divided by the rent of the year it is spent. <b>The rent after selling is the family-sized one</b>, '
        'not the one-bedroom the renting scenario moves to, because the household that sells has not moved.'),
    callout_housing=(
        '<strong>In the economies where prices rise fastest (%(econFast)s), selling the home in %(hlast)s buys '
        '%(coverFastLast)s years of rent; in %(hyears0)s it would have bought %(coverFastFirst)s.</strong> '
        'The fall is entirely the collateral being held in cash terms. If land in 八王子市 rises with prices, '
        'this understates what the home is worth, which is the safe direction; if land there falls, it overstates it.'),

    h_diy='Running it on your own conditions',
    p_diy=(
        'The simulator and the full-combination run are in the repository, and nothing in them is specific to this household. '
        'Three things change:'),
    diy_steps=[
        'Describe the household. Income, spending, housing, loans, pension and so on are one TSV file each under '
        '<code>data/controllable/</code>; the manifest <code>projects/base.tsv</code> lists which file fills which slot. '
        'Replace the figures with your own. The README in that directory says where each of the example figures came from.',
        'Write the options. Each option is another TSV file for the same slot; the ones used here are under '
        '<code>data/controllable/scenario/</code> (decisions) and <code>data/environment/scenario/</code> (economies, crashes).',
        'List the conditions. <code>Axes()</code> in <code>tools/sweep/main.go</code> names each condition, its options, '
        'and which file each option swaps in. Add, remove or rename conditions there. The number of plans to run '
        'is the product of the option counts.',
    ],
    p_diy_run=(
        'Then run the three commands in the footer. The sweep runs one plan per combination on every core; '
        'the two scripts produce this page from the results.'),
    foot_gen=(
        'Generated by <code>tools/sweep</code>, which passes overrides to <code>plan.Build</code> instead of '
        'writing %(n)s manifests. Results are in <code>out/sweep/cells.tsv</code>. Reproduce with '
        '<code>go run ./tools/sweep &amp;&amp; python3 examples/analyse.py &amp;&amp; python3 examples/build.py</code>.'),
    foot_caveat=(
        '<strong>The household in this report does not exist.</strong> Nobody lives on exactly these figures, '
        'but each figure comes from a published statistic. The salary comes from the 民間給与実態統計調査 by age band, '
        'the spouse\'s pay from the 賃金構造基本統計調査 hourly rate for part-time women, the spending from the '
        '家計調査 for the head\'s age band, and the rent and purchase price from the going rate for a 70㎡ flat in '
        '八王子市. Each source is cited in <code>data/controllable/README.md</code>. Only the household\'s '
        'makeup is invented; the decisions, the statutory rules and the economic projections are real. All amounts are in real terms, '
        'discounted to the plan\'s first year. Pension levels follow the 2024 財政検証. The crash size is a round figure '
        'chosen by hand rather than derived from any particular portfolio.'),
)

TEXT['ja'] = dict(
    lang='ja',
    kicker='lifeplan / 5つの条件の全組み合わせ / サンプル',
    title='この世帯がたどりうる%(n)s通りのシナリオ',
    lede=(
        '架空の世帯について、世帯が決められる3つの選択肢と世帯には選べない2つの外部条件のすべての組み合わせで%(n)s通りのシナリオを作成した。'
        '100歳時点で資産が残るシナリオもあれば資産が枯渇するシナリオもある。'
        '全通りの組み合わせを計算する目的は、<strong>どの条件がどの程度資産の推移を左右するか</strong>を見ることにある。'),

    h_axes='どんな条件を想定したか',
    p_axes='条件のすべての組み合わせを計算することで、とるべき選択肢がわかる。',
    p_method=(
        '誰にも制御できない条件（物価、賃金の伸び、運用の成績、暴落の時期）と意思決定（住まい、働き方、支出、子の進路、年金の受け取り方）の2つがある。'
        'この組み合わせごとにシナリオを計算すればどの意思決定が重要かを判断できる。'),
    th_axis='条件', th_chosen='世帯が選べるか', th_what='選択肢の内容',
    axes=[
        ('経済', 'いいえ', '公表されている7つの経済見通し'),
        ('金融危機', 'いいえ', '暴落なし、5つの年のいずれかで−20%%の下落、または2年連続の下落'),
        ('生活費', 'はい', '世帯主の年齢階級に対応する家計調査の平均、またはその±月4万円'),
        ('年金受給開始', 'はい', '原案どおり（70歳）、または夫婦とも65歳、70歳、75歳'),
        ('住まい', 'はい', '2023年に70㎡の分譲を購入、または生涯賃貸で子の独立後は1LDKへ移る'),
    ],
    cap_axes=(
        '7 × 7 × 3 × 4 × 2 = <b>%(n)s</b>通り。'
        '条件を1つ増やすごとに計算するプランの本数は掛け算で増えるので、条件をどう選ぶかがそのまま計算時間を決める。'),


    h_eta='どの条件が結果を動かすか',
    sub_eta='各条件がばらつきに占める割合（説明された分散、η²）',
    p_eta=(
        'η²は大雑把にはその選択肢の重大さである。'
        '厳密には、ある条件を1つの選択肢に固定したときに消えるばらつきを意味する。'),
    th_share='ばらつきに占める割合',
    cap_eta=(
        'オレンジは世帯が選べない条件、青は世帯が決められる条件。'
        '5つの条件それぞれ単独の効果を合計するとばらつきの<b>%(etaMain)s</b>を説明する。'
        '残りは条件どうしが組み合わさって生じる効果で、ある条件の効き方が別の条件の選択肢によって変わる分である。'),

    h_econ='経済は選べない',
    sub_econ='経済ごとに、他の4つの条件のすべての組み合わせを集計',
    th_econ='経済', th_worst='最悪', th_median='中央値', th_best='最良', th_ruin='破綻率',
    cap_econ='単位は億円、現在価値。網掛けの行がプラン原案の前提とする経済。',
    callout_econ_all=(
        '<strong>もっとも厳しい経済では、すべての組み合わせが破綻する。</strong>'
        '世帯の意思決定をどう設定しても、そこでは存続できない。'
        'これがこの計算のもっとも有用な出力である。'
        '問題が世帯の選択の外にあることを示している。'),
    callout_econ_most=(
        '<strong>もっとも厳しい経済（%(worstEcon)s）では、組み合わせの%(worstRuin)sが破綻する。</strong>'
        'そこで存続できるのは、世帯の意思決定をもっとも有利に設定した組み合わせだけである。'
        'これがこの計算のもっとも有用な出力である。'
        '問題のどれだけが世帯の選択の外にあるかを示している。'),

    h_dials='意思決定を1つずつ見る',
    sub_dials='各選択肢の純資産中央値と破綻率。他の条件のすべての組み合わせを集計',
    th_housing='住まい', th_living='生活費', th_pension='年金受給開始', th_crash='暴落', th_ruin_s='破綻率',
    cap_housing_bigger='住まいは世帯が決められる意思決定の中で最大で、経済よりも結果を動かす。',
    cap_housing_smaller='住まいは世帯が決められる意思決定の中で最大である。経済ほどではないが、他のどの意思決定よりも結果を動かす。',
    cap_living='支出も結果を動かすが、住まいほどではない。',
    cap_pension='年金の繰り下げは中央値を押し上げるが、受給を待つ年数は貯蓄の取り崩しでしのぐことになる。',
    cap_crash='同じ下落率でも、いつ起きるか、そのとき何をどれだけ保有しているかで損失額は変わる。',
    p_dials_note=(
        '表の行どうしがほとんど変わらない意思決定は、この世帯ではほとんど結果を左右しない。'
        '%(smallDials)sは、ばらつきの1%%未満しか占めない。'
        'これも結果の1つである。世帯がどの問いに悩むのをやめてよく、どの問い（%(topDial)s）に時間を使うべきかを示している。'),

    h_heat='住まいと生活費の組み合わせ',
    sub_heat='各行は、他の3つの条件のすべての組み合わせを集計',
    cap_heat='各行の中央値と破綻率は、それらの組み合わせ全体から求めたもの。',

    h_resort='破産後の行動',
    sub_resort='資産が枯渇すると、自宅を売却して賃貸に移るか自宅を担保に借りることになる',
    p_resort=(
        'どちらの手段も先に住宅ローンを完済している必要がある。'
        'すでに担保に入れた家を再び担保にはできないため売却する場合は売却代金からローンを清算しなければならない。'
        '<strong>両方の選択肢を試算し有利な方を採用している。</strong>'),
    th_measure='手段', th_times='採用回数',
    cap_resortM=(
        '<b>%(resortUsed)s</b>通りの組み合わせで採用された。'
        '必要になった年は最も早くて<b>%(resortMin)s</b>年、中央値は<b>%(resortMed)s</b>年、最も遅くて<b>%(resortMax)s</b>年。'),
    th_cells_used='採用した組み合わせの数', th_share_s='割合',
    cap_resortE=(
        '自宅の現金化が起きた経済は%(resortEcons)sだけである。'
        '<b>自宅を手放すかどうかは、世帯が何かを選ぶより先に、経済によって決まっている。</b>'),


    h_housing='家賃と担保評価額はどう動くか',
    sub_housing='経済ごとに、賃貸の家賃と、自宅を担保に入れたときの評価額を年で追う',
    p_housing=(
        '家賃は物価に連れて上がる。'
        '担保に入れられる評価額は固定資産税の納税通知書から読んだ土地の評価額で、計算の全期間にわたって名目のまま据え置いている。'
        'この2つは同じ向きに動かないので、自宅を現金に換えるのが遅いほど、買える家賃の年数は少なくなる。'),
    cap_rent=(
        '<b>家賃（名目）。</b>単位は円/年、その年の価格。実質では%(moveYear)s年の1LDKへの転居まで一定なので、'
        'この段以外の動きはすべて物価による。'
        '7つの経済が作る物価の道筋は<b>%(paths)s</b>通りしかなく、同じ道筋の経済は同じ行になる。'),
    cap_real=(
        '<b>担保評価額（実質）。</b>同じ%(collateral)s万円を、プラン初年度の価格に割り戻したもの。'
        '土地について何かが変わったわけではない。物価が上がるあいだ名目で据え置いているので、実質では下がる。'),
    cap_cover=(
        '<b>売却代金で払える家賃の年数。</b>担保評価額%(collateral)s万円のうち、売却して手元に残るのは%(proceedRate)sの%(proceeds)s万円である。'
        'これをその年の家賃で割った。'
        '<b>売却後に払う家賃は子が居たころと同じ広さのもの</b>で、賃貸シナリオが移る1LDKではない。'
        '売却する世帯は転居していないからである。'),
    callout_housing=(
        '<strong>物価がもっとも速く上がる経済（%(econFast)s）では、%(hlast)s年に自宅を売っても家賃%(coverFastLast)s年分にしかならない。'
        '%(hyears0)s年なら%(coverFastFirst)s年分だった。</strong>'
        'この目減りは担保を名目で据え置いていることだけから来ている。'
        '八王子市の地価が物価に連れて上がるなら、これは自宅を安く見積もりすぎており、安全側である。'
        '逆に地価が下がるなら、高く見積もっていることになる。'),

    h_diy='自分の条件で試算する方法',
    p_diy=(
        'シミュレータも全組み合わせの計算もリポジトリに含まれておりこの世帯に固有の部分はない。'
        '差し替えるのは次の3つである。'),
    diy_steps=[
        '世帯を記述する。収入、支出、住まい、ローン、年金などは<code>data/controllable/</code>の下にTSVファイルが1つずつあり、'
        'マニフェスト<code>projects/base.tsv</code>がどのファイルをどの欄に入れるかを列挙している。数字を自分のものに置き換える。'
        'サンプルの数字の出典は、そのディレクトリのREADMEに書いてある。',
        '選択肢を書く。選択肢は同じ欄に入れる別のTSVファイルである。ここで使ったものは<code>data/controllable/scenario/</code>（意思決定）と'
        '<code>data/environment/scenario/</code>（経済、暴落）にある。',
        '条件を列挙する。<code>tools/sweep/main.go</code>の<code>Axes()</code>が、条件の名前、その選択肢、'
        '選択肢ごとに差し替えるファイルを並べている。条件の追加、削除、改名はここで行う。計算するプランの本数は選択肢の数の積になる。',
    ],
    p_diy_run=(
        'そのうえで末尾の3つのコマンドを実行する。全組み合わせの計算は全コアを使って組み合わせごとにプランを1本ずつ計算し、'
        '残り2つのスクリプトが結果からこのページを生成する。'),
    foot_gen=(
        '<code>tools/sweep</code>が生成した。%(n)s本のマニフェストを書き出す代わりに、'
        '<code>plan.Build</code>に上書き設定を渡している。'
        '結果は<code>out/sweep/cells.tsv</code>にある。再現するには'
        '<code>go run ./tools/sweep &amp;&amp; python3 examples/analyse.py &amp;&amp; python3 examples/build.py</code>を実行する。'),
    foot_caveat=(
        '<strong>この報告書の世帯は実在しない。</strong>この数字どおりに暮らしている人はいないが、'
        '数字はそれぞれ公開統計から取っている。'
        '給与は民間給与実態統計調査の年齢階級別の値、配偶者の給与は賃金構造基本統計調査のパート女性の時給、'
        '生活費は世帯主の年齢階級に対応する家計調査の値、家賃と購入価格は八王子市の70㎡分譲の相場から取っている。'
        'それぞれの出典は<code>data/controllable/README.md</code>に記載した。'
        '架空なのは世帯の構成だけで、意思決定の設定、法定の制度、経済見通しは実在のものである。'
        '金額はすべて実質値で、プラン初年度の価値に割り引いている。年金水準は2024年財政検証に従う。'
        '暴落の下落率はきりのよい数字として手で選んだもので、特定の資産構成から導いた値ではない。'),
)

# ---------------------------------------------------------------- html ---
CSS = '''
:root{color-scheme:light;--plane:#f3f4f5;--surface:#fcfcfb;--ink:#0b0b0b;--ink2:#52514e;--muted:#7d7c76;
--rule:rgba(11,11,11,.12);--s1:#2a78d6;--s2:#eb6834;--crit:#d03b3b;--band:rgba(42,120,214,.07)}
@media (prefers-color-scheme:dark){:root:not([data-theme="light"]){color-scheme:dark;--plane:#0c0c0c;--surface:#1a1a19;
--ink:#fff;--ink2:#c3c2b7;--muted:#898781;--rule:rgba(255,255,255,.14);--s1:#3987e5;--s2:#d95926;--crit:#d03b3b;--band:rgba(57,135,229,.10)}}
:root[data-theme="dark"]{color-scheme:dark;--plane:#0c0c0c;--surface:#1a1a19;--ink:#fff;--ink2:#c3c2b7;--muted:#898781;
--rule:rgba(255,255,255,.14);--s1:#3987e5;--s2:#d95926;--crit:#d03b3b;--band:rgba(57,135,229,.10)}
*{box-sizing:border-box}
body{margin:0;background:var(--plane);color:var(--ink);font:16px/1.75 -apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif}
.wrap{max-width:1000px;margin:0 auto;padding:0 24px 96px}
header{padding:64px 0 32px}.kicker{font-size:.75rem;letter-spacing:.2em;color:var(--muted);margin:0 0 18px}
h1{font-size:clamp(2rem,5vw,3.2rem);line-height:1.2;margin:0 0 18px;font-weight:600}
.lede{font-size:1.05rem;color:var(--ink2);margin:0;max-width:38em}
.tiles{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:1px;background:var(--rule);border:1px solid var(--rule);margin:40px 0 0}
@media(min-width:760px){.tiles{grid-template-columns:repeat(4,minmax(0,1fr))}}
.tile{background:var(--surface);padding:18px 20px}
.tile .v{font-size:1.9rem;line-height:1.1;font-weight:600;display:block;font-variant-numeric:tabular-nums}
.tile .k{font-size:.8rem;color:var(--ink2);margin-top:6px;display:block;line-height:1.5}
.tile.alarm .v{color:var(--crit)}
section{padding:48px 0 0}h2{font-size:1.5rem;margin:0 0 6px;font-weight:600}
h2+.sub{color:var(--muted);font-size:.85rem;margin:0 0 20px}
p{margin:0 0 1.1em}hr{height:1px;background:var(--rule);border:0;margin:48px 0 0}
figure{margin:26px 0 30px;background:var(--surface);border:1px solid var(--rule);padding:20px}
figcaption{font-size:.82rem;color:var(--ink2);margin-top:12px;line-height:1.65}
.scroll{overflow-x:auto}
table{border-collapse:collapse;width:100%;font-size:.88rem}
th,td{padding:8px 11px;text-align:left;border-bottom:1px solid var(--rule);vertical-align:baseline}
thead th{font-size:.76rem;color:var(--muted);font-weight:600;white-space:nowrap}
td.n,th.n{text-align:right;font-variant-numeric:tabular-nums}
tr.base td{background:var(--band)}
td.k{white-space:nowrap;font-variant-numeric:tabular-nums;color:var(--ink2)}
td.b{width:520px}td.b span{display:inline-block;height:12px;vertical-align:middle;border-radius:2px}
td.b .r{background:var(--crit)}td.b .s{background:var(--s1)}
td.b .env{background:var(--s2)}td.b .dial{background:var(--s1)}
td.rr{color:var(--crit)}
.callout{border-left:3px solid var(--s1);padding:2px 0 2px 18px;margin:26px 0}
.callout.warn{border-left-color:var(--crit)}
code{font-family:ui-monospace,SFMono-Regular,Menlo,monospace;background:var(--band);padding:.1em .35em;border-radius:3px;font-size:.92em}
ol{padding-left:1.4em}ol li{margin:0 0 .6em}
footer{margin-top:64px;padding-top:22px;border-top:1px solid var(--rule);font-size:.82rem;color:var(--muted)}
'''

SKELETON = '''<!-- generated by tools/sweep + examples/build.py; do not edit by hand -->
<title>{title}</title>
<style>{css}</style>
<div class="wrap" lang="{lang}">
<header>
<p class="kicker">{kicker}</p>
<h1>{title}</h1>
<p class="lede">{lede}</p>
</header>
<hr>
<section>
<h2>{h_axes}</h2>
<p>{p_axes}</p>
<p>{p_method}</p>
<figure><div class="scroll"><table>
<thead><tr><th>{th_axis}</th><th>{th_chosen}</th><th>{th_what}</th></tr></thead>
<tbody>{axes}</tbody></table></div>
<figcaption>{cap_axes}</figcaption></figure>
</section>
<hr>
<section>
<h2>{h_eta}</h2>
<p class="sub">{sub_eta}</p>
<p>{p_eta}</p>
<figure><div class="scroll"><table>
<thead><tr><th>{th_axis}</th><th class="b">{th_share}</th><th class="n">η²</th></tr></thead>
<tbody>{eta}</tbody></table></div>
<figcaption>{cap_eta}</figcaption></figure>
</section>
<hr>
<section>
<h2>{h_econ}</h2>
<p class="sub">{sub_econ}</p>
<figure><div class="scroll"><table>
<thead><tr><th>{th_econ}</th><th class="n">{th_worst}</th><th class="n">{th_median}</th><th class="n">{th_best}</th><th class="n">{th_ruin}</th></tr></thead>
<tbody>{econ}</tbody></table></div>
<figcaption>{cap_econ}</figcaption></figure>
<div class="callout warn">{callout_econ}</div>
</section>
<hr>
<section>
<h2>{h_dials}</h2>
<p class="sub">{sub_dials}</p>
<figure><div class="scroll"><table><thead><tr><th>{th_housing}</th><th class="n">{th_median}</th><th class="n">{th_ruin_s}</th></tr></thead><tbody>{dial_housing}</tbody></table></div>
<figcaption>{cap_housing}</figcaption></figure>
<figure><div class="scroll"><table><thead><tr><th>{th_living}</th><th class="n">{th_median}</th><th class="n">{th_ruin_s}</th></tr></thead><tbody>{dial_living}</tbody></table></div>
<figcaption>{cap_living}</figcaption></figure>
<figure><div class="scroll"><table><thead><tr><th>{th_pension}</th><th class="n">{th_median}</th><th class="n">{th_ruin_s}</th></tr></thead><tbody>{dial_pension}</tbody></table></div>
<figcaption>{cap_pension}</figcaption></figure>
<figure><div class="scroll"><table><thead><tr><th>{th_crash}</th><th class="n">{th_median}</th><th class="n">{th_ruin_s}</th></tr></thead><tbody>{dial_crisis}</tbody></table></div>
<figcaption>{cap_crash}</figcaption></figure>
<p>{p_dials_note}</p>
</section>
<hr>
<section>
<h2>{h_heat}</h2>
<p class="sub">{sub_heat}</p>
<figure><div class="scroll"><table>
<thead><tr><th>{th_housing}</th><th>{th_living}</th><th class="n">{th_median}</th><th class="n">{th_ruin}</th></tr></thead>
<tbody>{heat}</tbody></table></div>
<figcaption>{cap_heat}</figcaption></figure>
</section>
<hr>
<section>
<h2>{h_resort}</h2>
<p class="sub">{sub_resort}</p>
<p>{p_resort}</p>
<figure><div class="scroll"><table><thead><tr><th>{th_measure}</th><th class="n">{th_times}</th></tr></thead><tbody>{resortM}</tbody></table></div>
<figcaption>{cap_resortM}</figcaption></figure>
<figure><div class="scroll"><table><thead><tr><th>{th_econ}</th><th class="n">{th_cells_used}</th><th class="n">{th_share_s}</th></tr></thead><tbody>{resortE}</tbody></table></div>
<figcaption>{cap_resortE}</figcaption></figure>
</section>
<hr>
<section>
<h2>{h_housing}</h2>
<p class="sub">{sub_housing}</p>
<p>{p_housing}</p>
<figure><div class="scroll"><table>
<thead><tr><th>{th_econ}</th>{hyearHeads}</tr></thead>
<tbody>{rentRows}</tbody></table></div>
<figcaption>{cap_rent}</figcaption></figure>
<figure><div class="scroll"><table>
<thead><tr><th>{th_econ}</th>{hyearHeads}</tr></thead>
<tbody>{realRows}</tbody></table></div>
<figcaption>{cap_real}</figcaption></figure>
<figure><div class="scroll"><table>
<thead><tr><th>{th_econ}</th>{hyearHeads}</tr></thead>
<tbody>{coverRows}</tbody></table></div>
<figcaption>{cap_cover}</figcaption></figure>
<div class="callout">{callout_housing}</div>
</section>
<hr>
<section>
<h2>{h_diy}</h2>
<p>{p_diy}</p>
<ol>{diy_steps}</ol>
<p>{p_diy_run}</p>
</section>
<footer>
<p>{foot_gen}</p>
<p>{foot_caveat}</p>
</footer>
</div>
'''


def tables():
    emx = max(e['v'] for e in D['eta'])
    eta = ''.join(
        '<tr><td class="k">%s</td><td class="b"><span class="%s" style="width:%dpx"></span></td><td class="n">%s</td></tr>'
        % (e['name'], 'env' if e['env'] else 'dial', bar(e['v'], emx, 460), '%.2f%%' % (e['v'] * 100))
        for e in D['eta'])
    econ = ''.join(
        '<tr%s><td>%s</td><td class="n">%s</td><td class="n">%s</td><td class="n">%s</td><td class="n">%s</td></tr>'
        % (' class="base"' if e['name'] == '成長型' else '', e['name'], oku(e['min']), oku(e['med']), oku(e['max']), pct(e['ruin']))
        for e in D['econ'])

    def dial(name):
        return ''.join('<tr><td>%s</td><td class="n">%s</td><td class="n">%s</td></tr>' % (r['lv'], oku(r['med']), pct(r['ruin']))
                       for r in D['dials'][name])
    heat = ''.join('<tr><td>%s</td><td>%s</td><td class="n">%s</td><td class="n">%s</td></tr>'
                   % (h['住まい'], h['生活費'], oku(h['med']), pct(h['ruin'])) for h in D['heat'])
    rm = ''.join('<tr><td>%s</td><td class="n">%s</td></tr>' % (m['name'], com(m['n'])) for m in res['byMeasure'])
    re_ = ''.join('<tr><td>%s</td><td class="n">%s</td><td class="n">%s</td></tr>'
                  % (e['name'], com(e['used']), pct(e['used'] / e['n'])) for e in res['byEconomy'])
    hyear_heads = ''.join('<th class="n">%d</th>' % y for y in hyears)
    rent_rows = ''.join(
        '<tr%s><td>%s</td>%s</tr>'
        % (' class="base"' if e['name'] == '成長型' else '', e['name'],
           ''.join('<td class="n">%s</td>' % com(r['rent']) for r in e['rows']))
        for e in hs['econ'])
    real_rows = ''.join(
        '<tr%s><td>%s</td>%s</tr>'
        % (' class="base"' if e['name'] == '成長型' else '', e['name'],
           ''.join('<td class="n">%s</td>' % com(int(round(hs['collateral'] / r['level']))) for r in e['rows']))
        for e in hs['econ'])
    cover_rows = ''.join(
        '<tr%s><td>%s</td>%s</tr>'
        % (' class="base"' if e['name'] == '成長型' else '', e['name'],
           ''.join('<td class="n">%s</td>' % yrs(r['cover']) for r in e['rows']))
        for e in hs['econ'])
    return dict(hyearHeads=hyear_heads, rentRows=rent_rows, coverRows=cover_rows, realRows=real_rows,
                eta=eta, econ=econ, heat=heat, resortM=rm, resortE=re_,
                dial_housing=dial('住まい'), dial_living=dial('生活費'),
                dial_pension=dial('年金受給開始'), dial_crisis=dial('金融危機'))


def render(lang):
    T = TEXT[lang]
    join = (lambda xs: '、'.join(xs)) if lang == 'ja' else (lambda xs: ', '.join(xs[:-1]) + ' and ' + xs[-1] if len(xs) > 1 else xs[0])
    nums = dict(
        n=com(D['n']),
        etaMain='%.1f%%' % (D['etaMainSum'] * 100), etaTop='%.0f%%' % (top['v'] * 100), topAxis=top['name'],
        resortUsed=com(res['used']), resortMin=ry.get('min', '—'), resortMed=ry.get('med', '—'), resortMax=ry.get('max', '—'),
        worstEcon=worst_econ['name'], worstRuin=pct(worst_econ['ruin']),
        resortEcons=join(resort_econ),
        topDial=top_dial['name'],
        smallVerb='accounts' if len([e for e in dials if e['v'] < 0.01]) == 1 else 'each account',
        smallDials=join([e['name'] for e in D['eta'] if not e['env'] and e['v'] < 0.01]),
        moveYear=MOVE_YEAR, paths=hs['paths'], hlast=hlast, hyears0=hyears[0],
        collateral=man(hs['collateral']), proceeds=man(hs['proceeds']),
        collateralYen=com(hs['collateral']), proceedsYen=com(hs['proceeds']),
        proceedRate='%d%%' % round(100 * hs['proceeds'] / hs['collateral']),
        econFast=join(econ_fast), econSlow=econ_slow,
        coverFastLast=yrs(fast), coverFastFirst=yrs(cover_first[econ_fast[0]]),
        coverSlowLast=yrs(cover_last[econ_slow]),
    )
    out = {}
    for k, v in T.items():
        if k == 'axes':
            out['axes'] = ''.join('<tr><td>%s</td><td>%s</td><td>%s</td></tr>' % (a, c, t % nums) for a, c, t in v)
        elif k == 'diy_steps':
            out[k] = ''.join('<li>%s</li>' % x for x in v)
        elif isinstance(v, str):
            out[k] = v % nums
    out['callout_econ'] = out['callout_econ_all'] if worst_all_fail else out['callout_econ_most']
    assert top_dial['name'] == '住まい', 'housing caption assumes 住まい is the largest dial; got %s' % top_dial['name']
    out['cap_housing'] = out['cap_housing_bigger'] if top_dial['v'] > econ_eta['v'] else out['cap_housing_smaller']
    out.update(nums)
    out.update(tables())
    out['css'] = CSS
    return SKELETON.format(**out)


if __name__ == '__main__':
    for lang, path in (('en', 'examples/sweep-report.html'), ('ja', 'examples/sweep-report_ja.html')):
        html = render(lang)
        io.open(path, 'w', encoding='utf-8').write(html)
        print('wrote %s (%d bytes)' % (path, len(html)))
