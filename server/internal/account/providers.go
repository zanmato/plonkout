package account

import (
	"encoding/hex"
	"strings"
)

// passkeyProviders names the passkey providers people actually use, by the
// AAGUID their authenticators report. Where the passkey lives says more than
// the device it was created from: a passkey made on a laptop by scanning a QR
// code with the phone lives in the phone's password manager.
//
// From the community list at github.com/passkeydeveloper/passkey-authenticator-aaguids.
var passkeyProviders = map[string]string{
	"ea9b8d664d011d213ce4b6b48cb575d4": "Google Password Manager",
	"fbfc3007154e4ecc8c0b6e020557d7bd": "iCloud Keychain",
	"dd4ec289e01d41c9bb8970fa845d4bf2": "iCloud Keychain",
	"adce000235bcc60a648b0b25f1f05503": "Chrome on Mac",
	"08987058cadc4b81b6e130de50dcbe96": "Windows Hello",
	"9ddd1817af5a4672a2b93e3dd95000a9": "Windows Hello",
	"6028b017b1d44c02b4b3afcdafc96bb2": "Windows Hello",
	"bada5566a7aa401fbd9645619a55120d": "1Password",
	"d548826e79b4db40a3d811116f7e8349": "Bitwarden",
	"531126d6e717415c93203d9aa6981239": "Dashlane",
	"53414d53554e47000000000000000000": "Samsung Pass",
	"50726f746f6e5061737350726f746f6e": "Proton Pass",
	"fdb141b25d84443e8a354698c205a502": "KeePassXC",
}

// passkeyName picks a passkey's name: the provider when its AAGUID is known,
// otherwise the name the browser suggested, which is only a guess from the
// device it runs on.
func passkeyName(aaguid []byte, suggested string) string {
	if provider, ok := passkeyProviders[strings.ToLower(hex.EncodeToString(aaguid))]; ok {
		return provider
	}
	return passkeyNameOr(suggested)
}
