#!/usr/bin/env bash

echo -n "Remove KUDO from cluster (y/n)? "
old_stty_cfg=$(stty -g)
stty raw -echo ; answer=$(head -c 1) ; stty "$old_stty_cfg"
if echo "$answer" | grep -iq "^y" ;then
  echo "Removing KUDO..."
  kubectl kudo init --dry-run -o yaml | kubectl delete -f -
  kubectl delete crd instances.kudo.dev operators.kudo.dev operatorversions.kudo.dev
fi
