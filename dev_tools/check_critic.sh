#!/bin/sh

FILE=$GOPATH/bin/gocritic
if [ -f $FILE ]; then
    :
else
    echo "go get -u github.com/go-critic/go-critic/..."
    go get -u github.com/go-critic/go-critic/...
fi

# Not using `gocritic check-project` becoz it doesn't work good
# gocritic check-project ..

# Use `gocritic check-package` instead
PACKAGES=$(go list ../... | grep -v '/vendor/' | grep '/application/' )
echo "$PACKAGES" > packages.log
while read -r package; do
    echo "checking package:" "$package"
    gocritic check "$package"
    echo "checked"
    echo ""
done < packages.log
rm packages.log
