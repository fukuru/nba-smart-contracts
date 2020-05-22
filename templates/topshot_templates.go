package templates

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/onflow/nba-smart-contracts/data"

	"github.com/onflow/flow-go-sdk"
)

// GenerateMintMomentScript generates a script to mint a new moment
// from a play-set combination
func GenerateMintMomentScript(tokenCodeAddr, recipientAddress flow.Address, setID, playID uint32) ([]byte, error) {
	template := `
		import TopShot from 0x%s

		transaction {
			let adminRef: &TopShot.Admin
		
			prepare(acct: AuthAccount) {
				self.adminRef = acct.borrow<&TopShot.Admin>(from: /storage/TopShotAdmin)!
			}
		
			execute {
				let setRef = self.adminRef.borrowSet(setID: %d)

				// Mint a new NFT
				let moment1 <- setRef.mintMoment(playID: %d)
				let recipient = getAccount(0x%s)
				// get the Collection reference for the receiver
				let receiverRef = recipient.getCapability(/public/MomentCollection)!.borrow<&{TopShot.MomentCollectionPublic}>()!
				// deposit the NFT in the receivers collection
				receiverRef.deposit(token: <-moment1)
			}
		}`
	return []byte(fmt.Sprintf(template, tokenCodeAddr.String(), setID, playID, recipientAddress.String())), nil
}

// GenerateBatchMintMomentScript mints multiple moments of the same play-set combination
func GenerateBatchMintMomentScript(tokenCodeAddr flow.Address, destinationAccount flow.Address, setID, playID uint32, quantity uint64) ([]byte, error) {
	template := `
		import TopShot from 0x%s

		transaction {
			let adminRef: &TopShot.Admin
		
			prepare(acct: AuthAccount) {
				self.adminRef = acct.borrow<&TopShot.Admin>(from: /storage/TopShotAdmin)!
			}
		
			execute {
				let setRef = self.adminRef.borrowSet(setID: %d)

				// Mint a new NFT
				let collection <- setRef.batchMintMoment(playID: %d, quantity: %d)
				let recipient = getAccount(0x%s)
				// get the Collection reference for the receiver
				let receiverRef = recipient.getCapability(/public/MomentCollection)!.borrow<&{TopShot.MomentCollectionPublic}>()!
				// deposit the NFT in the receivers collection
				receiverRef.batchDeposit(tokens: <-collection)
			}
		}`

	return []byte(fmt.Sprintf(template, tokenCodeAddr.String(), setID, playID, quantity, destinationAccount)), nil
}

// GenerateAddPlayToSetScript adds a play to a set
// so that moments can be minted from the combo
func GenerateAddPlayToSetScript(tokenCodeAddr flow.Address, setID, playID uint32) ([]byte, error) {
	template := `
		import TopShot from 0x%s
		
		transaction {

			prepare(acct: AuthAccount) {
				let admin = acct.borrow<&TopShot.Admin>(from: /storage/TopShotAdmin)!
				let setRef = admin.borrowSet(setID: %d)
				setRef.addPlay(playID: %d)
			}
		}`
	return []byte(fmt.Sprintf(template, tokenCodeAddr.String(), setID, playID)), nil
}

// GenerateAddPlaysToSetScript adds multiple plays to a set
func GenerateAddPlaysToSetScript(tokenCodeAddr flow.Address, setID uint32, playIDs []uint32) ([]byte, error) {
	template := `
		import TopShot from 0x%s
		
		transaction {

			prepare(acct: AuthAccount) {
				let admin = acct.borrow<&TopShot.Admin>(from: /storage/TopShotAdmin)!
				let setRef = admin.borrowSet(setID: %d)
				setRef.addPlays(playIDs: %s)
			}
		}`
	return []byte(fmt.Sprintf(template, tokenCodeAddr.String(), setID, uint32ToCadenceArr(playIDs))), nil
}

func uint32ToCadenceArr(nums []uint32) []byte {
	var s string
	for _, n := range nums {
		s += fmt.Sprintf("UInt32(%d), ", n)
	}
	// slice the last 2 characters off as that's the comma and the whitespace
	return []byte("[" + s[:len(s)-2] + "]")
}

// GenerateMintPlayScript creates a new play data struct
// and initializes it with metadata
func GenerateMintPlayScript(tokenCodeAddr flow.Address, metadata data.PlayMetadata) ([]byte, error) {
	metadata = data.PlayMetadata{
		FullName: "testcase testlofsky",
	}
	md, err := json.Marshal(metadata)
	if err != nil {
		return nil, err
	}
	template := `
		import TopShot from 0x%s
		
		transaction {
			prepare(acct: AuthAccount) {
				let admin = acct.borrow<&TopShot.Admin>(from: /storage/TopShotAdmin)
					?? panic("No admin resource in storage")
				admin.createPlay(metadata: %s)
			}
		}`
	return []byte(fmt.Sprintf(template, tokenCodeAddr.String(), string(md))), nil
}

// GenerateMintSetScript creates a new Set struct and initializes its metadata
func GenerateMintSetScript(tokenCodeAddr flow.Address, name string) ([]byte, error) {
	template := `
		import TopShot from 0x%s

		transaction {
			prepare(acct: AuthAccount) {
				let admin = acct.borrow<&TopShot.Admin>(from: /storage/TopShotAdmin)!
				admin.createSet(name: "%s")
			}
		}`
	return []byte(fmt.Sprintf(template, tokenCodeAddr.String(), name)), nil
}

// GenerateFulfillPackScript creates a script that fulfulls a pack
func GenerateFulfillPackScript(tokenCodeAddr flow.Address, destinationAccount flow.Address, momentIDs []uint64) []byte {
	template := `
		import TopShot from 0x%s

		transaction {
			prepare(acct: AuthAccount) {
				let recipient = getAccount(0x%s)
				let receiverRef = recipient.getCapability(/public/MomentCollection)!.borrow<&{TopShot.MomentCollectionPublic}>()!
				let momentIDs = [%s]
				let collection <- acct.borrow<&TopShot.Collection>(from: /storage/MomentCollection)!.batchWithdraw(ids: momentIDs)
				receiverRef.batchDeposit(tokens: <-collection)
			}
		}`

	// Stringify moment IDs
	momentIDList := ""
	for _, momentID := range momentIDs {
		id := strconv.Itoa(int(momentID))
		momentIDList = momentIDList + `UInt64(` + id + `), `
	}
	// Remove comma and space from last entry
	if idListLen := len(momentIDList); idListLen > 2 {
		momentIDList = momentIDList[:len(momentIDList)-2]
	}

	return []byte(fmt.Sprintf(template, tokenCodeAddr.String(), destinationAccount.String(), momentIDList))
}

// GenerateTransferAdminScript generates a script to create and admin capability
// and transfer it to another account's admin receiver
func GenerateTransferAdminScript(topshotAddr, adminReceiverAddr flow.Address) ([]byte, error) {
	template := `
		import TopShot from 0x%s
		import TopshotAdminReceiver from 0x%s
		
		transaction {
		
			prepare(acct: AuthAccount) {
				let admin <- acct.load<@TopShot.Admin>(from: /storage/TopShotAdmin)
					?? panic("No topshot admin in storage")

				TopshotAdminReceiver.storeAdmin(newAdmin: <-admin)
			}
		}`
	return []byte(fmt.Sprintf(template, topshotAddr.String(), adminReceiverAddr.String())), nil
}
