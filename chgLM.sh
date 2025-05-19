#!/bin/bash

nfn=`ls -tr SRGSE5M.2025*|tail -1`; echo $nfn
nbe=$(basename "$nfn");echo $nbe

systemctl --user stop SRGSE5M
rm SRGSE5M
ln -s ./$nbe SRGSE5M
ls -l
systemctl --user start SRGSE5M
systemctl --user status SRGSE5M
