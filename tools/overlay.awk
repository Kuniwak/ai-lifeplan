# 差分を土台に重ねる。鍵は 1 列目。
#
# 見出しが一致しないと止める——列が入れ替わっていれば、重ねたつもりの値が
# 別の列に入る。差分にしかない鍵も止める——土台にその行が無いということは、
# 差分の側が古いか、鍵を打ち間違えたということである。
FNR == NR {
    if (FNR == 1) { head = $0; next }
    diff[$1] = $0
    seen[$1] = 0
    next
}
FNR == 1 {
    if ($0 != head) {
        print "overlay: 見出しが違う。差分と土台は同じ列でなければならない" > "/dev/stderr"
        exit 1
    }
    print
    next
}
{
    if ($1 in diff) { print diff[$1]; seen[$1] = 1 } else { print }
}
END {
    for (k in diff) {
        if (seen[k] != 1) {
            print "overlay: 鍵 " k " が土台に無い。差分は土台の行を置き換えるものである" > "/dev/stderr"
            exit 1
        }
    }
}
