build:
	rm -rf idler
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build -o idler .
	rm -rf ./idler.app
	mkdir -p idler.app/Contents/{MacOS,Resources}
	cp idler idler.app/Contents/MacOS/idler
	cp idler.icns idler.app/Contents/Resources/idler.icns
	cp Info.plist idler.app/Contents/Info.plist
	cp -R idler.app ~/Applications/
	rm -rf idler.app idler

