#!/bin/bash
#set -v
set -e
go build .
for d in ./data/*.in; do
    current_best=$(grep $(basename ${d}) ./data/best_scores | cut -d: -f2 )
    score=$(./fsp2 < ${d} | head -1)
    diff=$(( ${score} - ${current_best} ))
    echo $(basename ${d}):${score} \( ${diff} \)
done
