package gopds

var (
	RootFeed OpdsFeedDB = OpdsFeedDB{OpdsCommon: &OpdsCommon{
		Id:    "urn:uuid:" + Uuidgen(),
		Title: "Catalog Root"},
		Type:    Nav,
		Desc: "Top level catalog",
		Entries: []string{"all"}}
	AllFeed OpdsFeedDB = OpdsFeedDB{OpdsCommon: &OpdsCommon{
		Id:    "urn:uuid:" + Uuidgen(),
		Title: "All Books"},
		Desc: "All books",
		Sort: SortTitle,
		Type: Acq}
)
