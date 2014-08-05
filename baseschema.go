package gopds

var (
	RootFeed OpdsFeedDB = OpdsFeedDB{OpdsCommon: &OpdsCommon{
		Id:    "urn:uuid:" + Uuidgen(),
		Title: "Catalog Root"},
		Type:    Nav,
		Entries: []string{"all"}}
	AllFeed OpdsFeedDB = OpdsFeedDB{OpdsCommon: &OpdsCommon{
		Id:    "urn:uuid:" + Uuidgen(),
		Title: "All Books"},
		Type: Acq,
		All:  true}
)
