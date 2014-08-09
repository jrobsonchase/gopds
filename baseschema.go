package gopds

var (
	RootFeed OpdsFeedDB = OpdsFeedDB{OpdsCommon: &OpdsCommon{
		Id:    "urn:uuid:" + Uuidgen(),
		Title: "Catalog Root",
		Name: "",
		Type:    Nav},
		Desc: "Top level catalog",
		Entries: []string{"all"}}
	AllFeed OpdsFeedDB = OpdsFeedDB{OpdsCommon: &OpdsCommon{
		Id:    "urn:uuid:" + Uuidgen(),
		Title: "All Books",
		Name: "all",
		Type: Acq},
		Desc: "All books",
		Sort: SortTitle}
)
