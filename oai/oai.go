package oai

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

type Header struct {
	Status     string   `xml:"status,attr"`
	Identifier string   `xml:"identifier"`
	DateStamp  string   `xml:"datestamp"`
	SetSpec    []string `xml:"setSpec"`
}

type Metadata struct {
	Body []byte `xml:",innerxml"`
}

type About struct {
	Body []byte `xml:",innerxml"`
}

type Record struct {
	Header   Header   `xml:"header"`
	Metadata Metadata `xml:"metadata"`
	About    About    `xml:"about"`
}

type ListIdentifiers struct {
	Headers         []Header `xml:"header"`
	ResumptionToken string   `xml:"resumptionToken"`
}

type ListRecords struct {
	Records         []Record `xml:"record"`
	ResumptionToken string   `xml:"resumptionToken"`
}

type GetRecord struct {
	Record Record `xml:"record"`
}

type RequestNode struct {
	Verb           string `xml:"verb,attr"`
	Set            string `xml:"set,attr"`
	MetadataPrefix string `xml:"metadataPrefix,attr"`
}

type OAIError struct {
	Code    string `xml:"code,attr"`
	Message string `xml:",chardata"`
}

type MetadataFormat struct {
	MetadataPrefix    string `xml:"metadataPrefix"`
	Schema            string `xml:"schema"`
	MetadataNamespace string `xml:"metadataNamespace"`
}

type ListMetadataFormats struct {
	MetadataFormat []MetadataFormat `xml:"metadataFormat"`
}

type Description struct {
	Body []byte `xml:",innerxml"`
}

type Set struct {
	SetSpec        string      `xml:"setSpec"`
	SetName        string      `xml:"setName"`
	SetDescription Description `xml:"setDescription"`
}

type ListSets struct {
	Set []Set `xml:"set"`
}

type Identify struct {
	RepositoryName    string        `xml:"repositoryName"`
	BaseURL           string        `xml:"baseURL"`
	ProtocolVersion   string        `xml:"protocolVersion"`
	AdminEmail        []string      `xml:"adminEmail"`
	EarliestDatestamp string        `xml:"earliestDatestamp"`
	DeletedRecord     string        `xml:"deletedRecord"`
	Granularity       string        `xml:"granularity"`
	Description       []Description `xml:"description"`
}

// The struct representation of an OAI-PMH XML response
type Response struct {
	ResponseDate string      `xml:"responseDate"`
	Request      RequestNode `xml:"request"`
	Error        OAIError    `xml:"error"`

	Identify            Identify            `xml:"Identify"`
	ListMetadataFormats ListMetadataFormats `xml:"ListMetadataFormats"`
	ListSets            ListSets            `xml:"ListSets"`
	GetRecord           GetRecord           `xml:"GetRecord"`
	ListIdentifiers     ListIdentifiers     `xml:"ListIdentifiers"`
	ListRecords         ListRecords         `xml:"ListRecords"`
}

// Formatter for Metadata content
func (md Metadata) GoString() string { return fmt.Sprintf("%s", md.Body) }

// Formatter for Description content
func (desc Description) GoString() string { return fmt.Sprintf("%s", desc.Body) }

// Formatter for About content
func (ab About) GoString() string { return fmt.Sprintf("%s", ab.Body) }

// Perform an HTTP GET request using the OAI Requests fields
// and return an OAI Response reference
func (req *Request) Perform() (oaiResponse *Response) {
	// Perform the GET request
	resp, err := http.Get(req.String())
	if err != nil {
		panic(err)
	}

	// Make sure the response body object will be closed after
	// reading all the content body's data
	defer resp.Body.Close()

	// Read all the data
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	// Unmarshall all the data
	err = xml.Unmarshal(body, &oaiResponse)
	if err != nil {
		panic(err)
	}

	return
}

// Represents a request URL and query string to an OAI-PMH service
type Request struct {
	BaseUrl, Set, MetadataPrefix, Verb, Identifier, ResumptionToken, From, Until string
}

// String representation of the OAI Request
func (req *Request) String() string {
	qs := []string{}

	add := func(name, value string) {
		if value != "" {
			qs = append(qs, name+"="+value)
		}
	}

	add("verb", req.Verb)
	add("set", req.Set)
	add("metadataPrefix", req.MetadataPrefix)
	add("resumptionToken", req.ResumptionToken)
	add("identifier", req.Identifier)
	add("from", req.From)
	add("until", req.Until)

	return strings.Join([]string{req.BaseUrl, "?", strings.Join(qs, "&")}, "")
}

// Perform a harvest of a complete OAI set, or simply one request
// call the batchCallback function argument with the OAI responses
func (req *Request) Harvest(batchCallback func(*Response)) {
	// Use Perform to get the OAI response
	oaiResponse := req.Perform()

	// Execute the callback function with the response
	batchCallback(oaiResponse)

	// Check for a resumptionToken
	hasResumptionToken, resumptionToken := oaiResponse.ResumptionToken()

	// Harvest further if there is a resumption token
	if hasResumptionToken == true {
		req.Set = ""
		req.MetadataPrefix = ""
		req.From = ""
		req.ResumptionToken = resumptionToken
		req.Harvest(batchCallback)
	}
}

// Determine the resumption token in this Response
func (resp *Response) ResumptionToken() (hasResumptionToken bool, resumptionToken string) {
	hasResumptionToken = false
	resumptionToken = ""
	if resp == nil {
		return
	}

	// First attempt to obtain a resumption token from a ListIdentifiers response
	resumptionToken = resp.ListIdentifiers.ResumptionToken

	// Then attempt to obtain a resumption token from a ListRecords response
	if resumptionToken == "" {
		resumptionToken = resp.ListRecords.ResumptionToken
	}

	// If a non-empty resumption token turned up it can safely inferred that...
	if resumptionToken != "" {
		hasResumptionToken = true
	}

	return
}

// Harvest the identifiers of a complete OAI set
// call the identifier callback function for each Header
func (req *Request) HarvestIdentifiers(callback func(*Header)) {
	req.Verb = "ListIdentifiers"
	req.Harvest(func(resp *Response) {
		headers := resp.ListIdentifiers.Headers
		for _, header := range headers {
			callback(&header)
		}
	})
}

// Harvest the identifiers of a complete OAI set
// call the identifier callback function for each Header
func (req *Request) HarvestRecords(callback func(*Record)) {
	req.Verb = "ListRecords"
	req.Harvest(func(resp *Response) {
		records := resp.ListRecords.Records
		for _, record := range records {
			callback(&record)
		}
	})
}

// Reads OAI PMH response XML from a file
func FromFile(filename string) (oaiResponse *Response) {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}

	// Unmarshall all the data
	err = xml.Unmarshal(bytes, &oaiResponse)
	if err != nil {
		panic(err)
	}

	return
}

// Harvest the identifiers of a complete OAI set
// send a reference of each Header to a channel
func (req *Request) ChannelHarvestIdentifiers(channels []chan *Header) {
	req.Verb = "ListIdentifiers"
	req.Harvest(func(resp *Response) {
		headers := resp.ListIdentifiers.Headers
		i := 0
		for _, header := range headers {
			channels[i] <- &header
			i++
			if i == len(channels) {
				i = 0
			}
		}

		// If there is no more resumption token, send nil to all
		// the channels to signal the harvest is done
		hasResumptionToken, _ := resp.ResumptionToken()
		if !hasResumptionToken {
			for _, channel := range channels {
				channel <- nil
			}
		}
	})
}
