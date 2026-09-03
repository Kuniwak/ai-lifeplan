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
d = D['dist']
res = D['resort']
ry = res.get('years', {})
W = 520


def oku(v): return ('+' if v >= 0 else '−') + ('%.2f' % abs(v))
def pct(v): return '%.1f%%' % (v * 100)
def com(v): return '{:,}'.format(v)
def bar(v, mx, w=W): return max(2, int(round(v / mx * w))) if mx else 2


# ---------------------------------------------------------------- facts --
top = D['eta'][0]                         # axis with the largest η²
dials = [e for e in D['eta'] if not e['env']]
top_dial = dials[0]                       # largest axis the household chooses
econ_eta = next(e for e in D['eta'] if e['name'] == '経済')
worst_econ = max(D['econ'], key=lambda e: e['ruin'])
house = {r['lv']: r for r in D['dials']['住まい']}
resort_econ = [e['name'] for e in res['byEconomy'] if e['used']]
econ_names = [e['name'] for e in D['econ']]
worst_all_fail = worst_econ['ruin'] >= 1.0

# ---------------------------------------------------------------- text ---
TEXT = {}

TEXT['en'] = dict(
    lang='en',
    kicker='lifeplan / every combination of five conditions / example',
    title='%(n)s ways this household could go',
    lede=(
        'Five conditions were given two to seven options each, and every combination of those options was run '
        'as a complete life plan carried to the year the earner turns 100. The final net worth ranges from %(min)s to %(max)s '
        'hundred-million yen. The point of running them all is to see <strong>which condition moves the result</strong>. '
        'The household is invented; see the note at the end.'),
    tile_n='combinations, each one run as a complete plan',
    tile_top='of the spread comes from %(topAxis)s, the condition with the most influence',
    tile_main='of the spread is explained by the five conditions acting separately',
    tile_resort='combinations had to turn the home into cash',

    h_axes='What was varied',
    sub_axes='One condition per row. The first option of each condition is what the original plan assumed.',
    p_axes=(
        'A scenario like "carry on as now" changes several things at once. A results row for it can only '
        'say which scenario was chosen, not which of its parts made the difference. Splitting the scenarios '
        'into separate conditions, each with a few options, and running every combination of options is what '
        'answers that question.'),
    p_method=(
        'Nothing about this depends on the household. Any life plan is a set of decisions made under conditions '
        'nobody controls. Write each decision and each outside condition as a short list of options, run the plan '
        'once per combination, and the results show which decision is worth deliberating over and which '
        'is not. A single simulator run says only whether one particular plan works; this says '
        'what changes the answer.'),
    th_axis='Condition', th_levels='Options', th_chosen='Household chooses?', th_what='What the options are',
    axes=[
        ('経済', 'no', 'seven published economic projections'),
        ('金融危機', 'no', 'no crash, a −20%% crash in one of five chosen years, or a crash in two consecutive years'),
        ('生活費', 'yes', 'the 家計調査 average for the head\'s age band, or that figure ±¥40,000 a month'),
        ('年金受給開始', 'yes', 'as in the plan (70), or 65, 70, or 75 for both'),
        ('住まい', 'yes', 'buy a 70㎡ flat in 2023, or rent the same size for life'),
    ],
    cap_axes=(
        '7 × 7 × 3 × 4 × 2 = <b>%(n)s</b> combinations. Each added condition multiplies the number of plans to run, '
        'so choosing the conditions is also choosing how long the whole run takes.'),

    h_dist='How widely the results spread',
    sub_dist='Net worth in the final year, in today\'s prices, in hundred-million yen',
    p_dist=(
        'In the simulator, assets stop at zero and never go negative. A plan that cannot pay a bill records '
        'the unpaid amount as a shortfall and keeps going. If the chart used assets alone, every ruined combination '
        'would sit at zero and the chart would go flat exactly where the differences matter most. '
        'So the measure used here is <strong>assets minus the accumulated shortfall</strong>, which can go below zero.'),
    th_band='Range', th_cells='Combinations', th_n='count', th_ruined='of which ruined',
    cap_dist=(
        'Minimum <b>%(min)s</b>, lower quartile <b>%(q1)s</b>, median <b>%(med)s</b>, upper quartile <b>%(q3)s</b>, '
        'maximum <b>%(max)s</b>. Blue combinations stayed solvent; red ones ran out of money on the way. '
        'The outcome worsens continuously from "just made it" to "ruined".'),

    h_eta='Which condition moves the result',
    sub_eta='Share of the spread each condition accounts for (variance explained, η²)',
    p_eta=(
        'η² is the share of the total spread that disappears if one condition is held fixed. If the five shares '
        'added up to 100%%, each condition would act on its own; whatever is missing is the part where conditions '
        'act together. '
        'Because every option of every condition appears in the same number of combinations, the spread can be '
        'split cleanly by condition. '
        'This is a more reliable measure of sensitivity than moving one factor at a time, because a one-at-a-time '
        'result depends on where the other factors happened to be parked.'),
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
        'each account for under 1%% of the spread. That is a result too. It says which questions the household can '
        'stop deliberating over, and which one (%(topDial)s) deserves the time.'),

    h_heat='Housing and living cost together',
    sub_heat='Each row covers every combination of the other three conditions',
    cap_heat='The median and ruin rate in each row are taken over those combinations.',

    h_resort='The home is the last resource',
    sub_resort='When a plan cannot pay, it either sells the home and rents, or borrows against it',
    p_resort=(
        'Both measures require the mortgage to be repaid first. A home already pledged cannot be pledged again, '
        'and selling means clearing the loan out of the proceeds. <strong>The plan evaluates both and takes '
        'whichever leaves the household better off.</strong>'),
    th_measure='Measure', th_times='times taken',
    cap_resortM=(
        'Taken in <b>%(resortUsed)s</b> cells. The earliest year it was needed was <b>%(resortMin)s</b>, '
        'the median year <b>%(resortMed)s</b>, and the latest <b>%(resortMax)s</b>.'),
    th_cells_used='combinations that needed it', th_share_s='share',
    cap_resortE=(
        'Only these economies ever force the home into cash: %(resortEcons)s. '
        '<b>Whether the home has to be sold is settled by the economy before the household chooses anything.</b>'),

    h_not='What this sweep does not answer',
    p_not_ruin=(
        '<strong>How likely ruin is.</strong> %(ruinN)s of the %(n)s combinations fail, which is %(ruinPct)s. '
        'That figure is the share of the options that were fed in, not a probability. '
        'Load the sweep with pessimistic economies and it rises; load it with optimistic ones and it falls. '
        'Turning it into a chance would require weighting each option by how likely it is, and nobody can do that '
        'accurately. The seven economies and '
        'the seven crash timings are counted equally here, and they are not equally likely. '
        '<b>Read which condition moves the result, not how many combinations fail.</b>'),
    p_not_alloc=(
        '<strong>Asset allocation by age.</strong> Contributions follow one fixed policy, and nothing here varies '
        'the mix of shares and bonds. Knowing that a crash hurts most just before drawdown does not say what to do about it.'),
    p_not_other='<strong>Care, job loss, inheritance.</strong> None of these was varied.',
    p_not_real='<strong>Anything about a real household.</strong> Every figure describes an invented one.',

    h_diy='Doing the same for your own household',
    p_diy=(
        'The simulator and the sweep are in the repository, and nothing in them is specific to this household. '
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
    title='この世帯がたどりうる%(n)s通り',
    lede=(
        '5つの条件にそれぞれ2〜7個の選択肢を用意し、選択肢のすべての組み合わせについて、'
        '稼ぎ手が100歳になる年までライフプランを1本ずつ通しで計算した。'
        '最終年の純資産は%(min)s億円から%(max)s億円まで散らばる。'
        '全通りを計算する目的は、<strong>どの条件が結果を動かすか</strong>を見ることにある。'
        'この世帯は架空のものである（末尾の注記を参照）。'),
    tile_n='通りの組み合わせ。それぞれを1本のプランとして計算',
    tile_top='のばらつきが、もっとも影響の大きい条件である%(topAxis)sに由来する',
    tile_main='のばらつきが、5つの条件それぞれ単独の効果で説明できる',
    tile_resort='通りで、自宅を現金化せざるを得なかった',

    h_axes='何を振ったか',
    sub_axes='1行が1つの条件。各条件の最初の選択肢がプラン原案の前提',
    p_axes=(
        '「今のまま続ける」のようなシナリオは、複数の要素を同時に変える。'
        'その結果の行からは、どのシナリオを選んだかは分かっても、その中のどの要素が効いたかは分からない。'
        'シナリオを条件ごとに分解してそれぞれに数個の選択肢を置き、選択肢のすべての組み合わせを計算すれば、その問いに答えられる。'),
    p_method=(
        'この方法はこの世帯に固有のものではない。'
        'どのライフプランも、誰にも制御できない条件のもとで下す意思決定の集まりである。'
        '意思決定と外部条件をそれぞれ数個の選択肢に書き下し、組み合わせごとにプランを1本ずつ計算すれば、'
        'どの意思決定に悩む価値があり、どれには無いかが結果に現れる。'
        'シミュレータを1回動かして分かるのは、その1本のプランが成り立つかどうかだけである。'
        '全組み合わせを計算すると、何が答えを変えるのかが分かる。'),
    th_axis='条件', th_levels='選択肢の数', th_chosen='世帯が選べるか', th_what='選択肢の内容',
    axes=[
        ('経済', 'いいえ', '公表されている7つの経済見通し'),
        ('金融危機', 'いいえ', '暴落なし、5つの年のいずれかで−20%%の下落、または2年連続の下落'),
        ('生活費', 'はい', '世帯主の年齢階級に対応する家計調査の平均、またはその±月4万円'),
        ('年金受給開始', 'はい', '原案どおり（70歳）、または夫婦とも65歳、70歳、75歳'),
        ('住まい', 'はい', '2023年に70㎡の分譲を購入、または同じ広さを生涯賃貸'),
    ],
    cap_axes=(
        '7 × 7 × 3 × 4 × 2 = <b>%(n)s</b>通り。'
        '条件を1つ増やすごとに計算するプランの本数は掛け算で増えるので、条件をどう選ぶかがそのまま計算時間を決める。'),

    h_dist='結果はどれだけ散らばるか',
    sub_dist='最終年の純資産。現在価値、単位は億円',
    p_dist=(
        'シミュレータでは資産は0で下げ止まり、マイナスにはならない。'
        '支払えないプランは不足額を「不足」として記録し、計算を続ける。'
        '資産だけをそのまま描くと、破綻した組み合わせはすべて0に張り付き、差がもっとも重要な部分で図が横ばいになってしまう。'
        'そこでここでは<strong>資産から累積不足額を引いた値</strong>を指標にする。この値はマイナスにもなりうる。'),
    th_band='範囲', th_cells='組み合わせの数', th_n='件数', th_ruined='うち破綻',
    cap_dist=(
        '最小<b>%(min)s</b>、第1四分位<b>%(q1)s</b>、中央値<b>%(med)s</b>、第3四分位<b>%(q3)s</b>、最大<b>%(max)s</b>。'
        '青は最後まで存続した組み合わせ、赤は途中で資金が尽きた組み合わせ。'
        '結果は「ぎりぎり存続」から「破綻」へ連続的に悪化する。'),

    h_eta='どの条件が結果を動かすか',
    sub_eta='各条件がばらつきに占める割合（説明された分散、η²）',
    p_eta=(
        'η²は、ある条件を1つの選択肢に固定したときに消えるばらつきの割合である。'
        '5つの割合の合計が100%%なら各条件は単独で効いていることになり、100%%に足りない分は条件どうしが組み合わさって効いている部分である。'
        'どの条件のどの選択肢も同じ回数ずつ組み合わせに現れるので、ばらつきを条件ごとにきれいに分解できる。'
        '1つの条件だけを動かして他を固定する方法では、他の条件をどの選択肢に固定したかで結果が変わってしまう。'
        '全組み合わせからの分解は、その影響を受けない分だけ感度の測り方として信頼できる。'),
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
        '%(smallDials)sは、それぞればらつきの1%%未満しか占めない。'
        'これも結果の1つである。世帯がどの問いに悩むのをやめてよく、どの問い（%(topDial)s）に時間を使うべきかを示している。'),

    h_heat='住まいと生活費の組み合わせ',
    sub_heat='各行は、他の3つの条件のすべての組み合わせを集計',
    cap_heat='各行の中央値と破綻率は、それらの組み合わせ全体から求めたもの。',

    h_resort='最後に残るのは自宅',
    sub_resort='プランが支払えなくなると、自宅を売却して賃貸に移るか、自宅を担保に借りる',
    p_resort=(
        'どちらの手段も、先に住宅ローンを完済している必要がある。'
        'すでに担保に入れた家を再び担保にはできず、売却する場合は売却代金からローンを清算しなければならない。'
        '<strong>プランは両方を試算し、世帯にとって有利な方を自動で採用する。</strong>'),
    th_measure='手段', th_times='採用回数',
    cap_resortM=(
        '<b>%(resortUsed)s</b>通りの組み合わせで採用された。'
        '必要になった年は最も早くて<b>%(resortMin)s</b>年、中央値は<b>%(resortMed)s</b>年、最も遅くて<b>%(resortMax)s</b>年。'),
    th_cells_used='採用した組み合わせの数', th_share_s='割合',
    cap_resortE=(
        '自宅の現金化が起きた経済は%(resortEcons)sだけである。'
        '<b>自宅を手放すかどうかは、世帯が何かを選ぶより先に、経済によって決まっている。</b>'),

    h_not='この計算では答えられないこと',
    p_not_ruin=(
        '<strong>破綻の確率。</strong>%(n)s通りのうち%(ruinN)s通り、割合にして%(ruinPct)sが破綻する。'
        'この数字は入力した選択肢の構成比であって、確率ではない。'
        '悲観的な経済を多く入れれば上がり、楽観的な経済を多く入れれば下がる。'
        '確率に読み替えるには各選択肢の起こりやすさで重み付けする必要があるが、それを正確に見積もれる人はいない。'
        'ここでは7つの経済と7つの暴落時期を同じ重みで数えているが、それらが同じ確率で起きるわけではない。'
        '<b>何通り破綻するかではなく、どの条件が結果を動かすかを読むこと。</b>'),
    p_not_alloc=(
        '<strong>年齢別の資産配分。</strong>拠出は単一の運用方針に固定しており、株式と債券の配分は振っていない。'
        '暴落は取り崩し直前がもっとも痛いと分かっても、それにどう備えるべきかまでは分からない。'),
    p_not_other='<strong>介護、失職、相続。</strong>どれも条件として振っていない。',
    p_not_real='<strong>実在する世帯について。</strong>ここに出てくる数字はすべて架空の世帯のものである。',

    h_diy='自分の世帯で同じことをするには',
    p_diy=(
        'シミュレータも全組み合わせの計算もリポジトリに含まれており、この世帯に固有の部分はない。'
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
<div class="tiles">
<div class="tile"><span class="v">{n}</span><span class="k">{tile_n}</span></div>
<div class="tile"><span class="v">{etaTop}</span><span class="k">{tile_top}</span></div>
<div class="tile"><span class="v">{etaMain}</span><span class="k">{tile_main}</span></div>
<div class="tile"><span class="v">{resortUsed}</span><span class="k">{tile_resort}</span></div>
</div>
<hr>
<section>
<h2>{h_axes}</h2>
<p class="sub">{sub_axes}</p>
<p>{p_axes}</p>
<p>{p_method}</p>
<figure><div class="scroll"><table>
<thead><tr><th>{th_axis}</th><th class="n">{th_levels}</th><th>{th_chosen}</th><th>{th_what}</th></tr></thead>
<tbody>{axes}</tbody></table></div>
<figcaption>{cap_axes}</figcaption></figure>
</section>
<hr>
<section>
<h2>{h_dist}</h2>
<p class="sub">{sub_dist}</p>
<p>{p_dist}</p>
<figure><div class="scroll"><table>
<thead><tr><th>{th_band}</th><th class="b">{th_cells}</th><th class="n">{th_n}</th><th class="n">{th_ruined}</th></tr></thead>
<tbody>{hist}</tbody></table></div>
<figcaption>{cap_dist}</figcaption></figure>
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
<h2>{h_not}</h2>
<p>{p_not_ruin}</p>
<p>{p_not_alloc}</p>
<p>{p_not_other}</p>
<p>{p_not_real}</p>
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
    mx = max(h['n'] for h in D['hist'])
    hist = ''.join(
        '<tr><td class="k">%s 〜 %s</td><td class="b"><span class="r" style="width:%dpx"></span><span class="s" style="width:%dpx"></span></td><td class="n">%s</td><td class="n rr">%s</td></tr>'
        % (oku(h['lo']), oku(h['lo'] + 0.5), bar(h['ruin'], mx),
           bar(h['n'] - h['ruin'], mx) if h['n'] - h['ruin'] else 0,
           com(h['n']), com(h['ruin']) if h['ruin'] else '—')
        for h in D['hist'])
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
    return dict(hist=hist, eta=eta, econ=econ, heat=heat, resortM=rm, resortE=re_,
                dial_housing=dial('住まい'), dial_living=dial('生活費'),
                dial_pension=dial('年金受給開始'), dial_crisis=dial('金融危機'))


def render(lang):
    T = TEXT[lang]
    join = (lambda xs: '、'.join(xs)) if lang == 'ja' else (lambda xs: ', '.join(xs[:-1]) + ' and ' + xs[-1] if len(xs) > 1 else xs[0])
    nums = dict(
        n=com(D['n']), ruinN=com(d['ruinN']), ruinPct=pct(d['ruin']),
        min=oku(d['min']), max=oku(d['max']), med=oku(d['med']), q1=oku(d['q1']), q3=oku(d['q3']),
        etaMain='%.1f%%' % (D['etaMainSum'] * 100), etaTop='%.0f%%' % (top['v'] * 100), topAxis=top['name'],
        resortUsed=com(res['used']), resortMin=ry.get('min', '—'), resortMed=ry.get('med', '—'), resortMax=ry.get('max', '—'),
        worstEcon=worst_econ['name'], worstRuin=pct(worst_econ['ruin']),
        resortEcons=join(resort_econ),
        topDial=top_dial['name'],
        smallDials=join([e['name'] for e in D['eta'] if not e['env'] and e['v'] < 0.01]),
    )
    levels = {a['name']: a['n'] for a in [
        {'name': '経済', 'n': len(D['econ'])},
        {'name': '金融危機', 'n': len(D['dials']['金融危機'])},
        {'name': '生活費', 'n': len(D['dials']['生活費'])},
        {'name': '年金受給開始', 'n': len(D['dials']['年金受給開始'])},
        {'name': '住まい', 'n': len(D['dials']['住まい'])}]}
    out = {}
    for k, v in T.items():
        if k == 'axes':
            out['axes'] = ''.join('<tr><td>%s</td><td class="n">%d</td><td>%s</td><td>%s</td></tr>'
                                  % (a, levels[a], c, t % nums) for a, c, t in v)
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
