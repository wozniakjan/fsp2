#!/bin/bash

for d in $(ls data/*.in); do
    current_best=$(grep $(basename ${d}) data/best_scores | cut -d: -f2)
    score=$(./fsp2 < ${d} | head -1)
    echo $(basename ${d}):${score} \( $(( ${score} - ${current_best} )) \)
done
