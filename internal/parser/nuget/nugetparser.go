package nuget

import "encoding/xml"

type NugetProjectPackages struct {
	XMLName xml.Name  `xml:"packages"`
	Package []Package `xml:"package"`
}

type NugetProjectCsproj struct {
	XMLName   xml.Name    `xml:"Project"`
	ItemGroup []ItemGroup `xml:"ItemGroup"`
}

type Package struct {
	Id      string `xml:"id,attr"`
	Version string `xml:"version,attr"`
}

type ItemGroup struct {
	PackageReference []PackageReference `xml:"PackageReference"`
}

type PackageReference struct {
	Include    string  `xml:"Include,attr"`
	Version    string  `xml:"Version,attr"`
	VersionTag Version `xml:"Version"`
}

type Version struct {
	Value string `xml:",chardata"`
}
