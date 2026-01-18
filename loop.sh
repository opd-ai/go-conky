#! /usr/bin/env sh
alias copilot="yes n | copilot --model claude-opus-4.5"
rm -f AUDIT.md
copilot -p "/delegate $(cat docs/COMPLIANCE.md)" --allow-all-tools --deny-tool sudo
devloop.sh
#rm AUDIT.md
