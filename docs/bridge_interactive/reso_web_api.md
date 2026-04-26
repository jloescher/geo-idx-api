RESO Web API
The Bridge platform allows you to query MLS data using the RESO Web API specification, which is based on OData. For more information about the specification, please visit the RESO website. The API has been RESO Platinum Certified to version 1.0.2.

Authorization
Your application should send an Authorization header with every HTTP request to the API:

Authorization: Bearer {token}
If you can't set headers, you can send the token in the access_token parameter in the query string of your request:

GET https://api.bridgedataoutput.com/api/v2/OData/{dataset_id}/{resource}?access_token={server_token}
API Response
By default, the RESO Web API will return 10 listings, regardless of the number of total records available. You may use the $top parameter to specify your request to return up to 200 listings at a time.

If there are more than 200 records available, you will need to paginate through the results.

If you wish to paginate through more than 10,000 listings you will need to use the dedicated replication endpoint.

All API responses besides metadata are returned in JSON format.

Paginating through results
If the total number of records that are returned by a query is greater than 200 (the maximum limit of a result set), then you will need to paginate through the results. This is done by incrementing the number of skipped records.

Return the next 200 records, ordered by ListPrice
https://api.bridgedataoutput.com/api/v2/OData/dataset_id/{resource}?access_token=access_token&$top=200&$skip=200&$orderby=ListPrice desc
Using Expand
Using the $expand operator allow you to include associated data from additional resources. For example, you are able to bring in more detail about the relevant office or member into the response payload for a property, without having to make a second or third API query to the other resources.

You are not able to use all query parameters on data you’ve expanded into your response.
If you are using the `$select` parameter in your query to limit the fields in the response payload, be sure to include the expanded field as well (eg, add the ListOffice field if you're expanding ListOffice)
Expand ListOffice in a Property query
https://api.bridgedataoutput.com/api/v2/OData/dataset_id/Property?access_token=access_token&$expand=ListOffice
Dataset Replication
While we encourage the use of the API to query the listing data as needed, there are use-cases where replication of the full dataset may be preferred.

To help with this, you are able to request data with the /replication endpoint. Whereas on-demand API requests can have a maximum of 200 results returned at once, with this endpoint the maximum $top parameter is 2,000 results.

The header of the response will contain a 'next' link. Results are returned ordered from oldest to newest, so by using the next link you are able to pull down all the available records to seed your data, and then continue using the next link at regular intervals to keep up to date.

Use the replication endpoint
https://api.bridgedataoutput.com/api/v2/OData/dataset_id/Property/replication?access_token=access_token
If you are trying to replicate anything more than 10,000 records you will need to use the replication endpoint.
Because of the potential payload size, this is only suitable for server to server requests.
This functionality is only available at the discretion of the MLS that has authorized data access.
Certain parameters like ‘skip’ and ‘orderby’ are not available with this endpoint.
To reduce payload size and improve performance we recommend using the `select` parameter to request only the fields you need.
BridgeModificationTimestamp is the best field to use for incremental updates as it represents the last modification in the Bridge system; ModificationTimestamp is ingested directly from the MLS system and is not a consistent proxy for modifications to the listing in the Bridge database.
Media
Rather than keeping it in a separate resource, Media is returned as an object directly on the Property record. Typically, it is the highest resolution media available from the MLS and is stored on our CDN. You may link directly to the CDN.

Metadata
You can request metadata that will return the fields and lookup values that have been made available to by the data provider. Metadata is returned in XML, according to the RESO spec.

Request metadata
https://api.bridgedataoutput.com/api/v2/OData/dataset_id/$metadata?access_token=access_token
Operators
You can constrain the result set of a resource by passing additional operators with your request. Valid operators include:

Operator	Description
eq	Equal
ne	Not equal
gt	Greater than
lt	Less than
ge	Greater than or equal
le	Less than or equal
and	Logical and
or	Logical or
not	Logical not
Parameters
The following query parameters may be passed to narrow down your results.

Name	Type	Description
access_token	string	Token to identify the application. This is always required.
Example request: return all datasets approved for an application
https://api.bridgedataoutput.com/api/v2/OData/DataSystem?access_token=access_token
ListingKey	string	The listing key, available on the /Properties resource.
return data relating to a listing where the ListingKey “12345”
https://api.bridgedataoutput.com/api/v2/OData/dataset_id/Properties(‘12345’)?access_token=access_token
MemberKey	string	The member key, available on the /Members resource.
Return data relating to a member where the memberKey is “12345
https://api.bridgedataoutput.com/api/v2/OData/dataset_id/Members('12345')?access_token=access_token
OfficeKey	string	The office key, available on the /Offices resource.
Return data relating to an office where the officeKey is “12345”
https://api.bridgedataoutput.com/api/v2/OData/dataset_id/Offices('12345')?access_token=access_token
$skip	number	Skips this number of results
Skip the first 10 records of a dataset
https://api.bridgedataoutput.com/api/v2/OData/dataset_id/Properties?access_token=access_token&$skip=10
$select	string	Select the fields to be returned
Only return the LivingArea field
https://api.bridgedataoutput.com/api/v2/OData/dataset_id/Properties?access_token=access_token&$select=LivingArea
$unselect	string	Select the fields to be exluded
Do not return the Media object
https://api.bridgedataoutput.com/api/v2/OData/dataset_id/Properties?access_token=access_token&$unselect=Media
$filter	string	Filter the results to be returned
Only return the listings where the ListPrice is greater than $100000
https://api.bridgedataoutput.com/api/v2/OData/dataset_id/Properties?access_token=access_token&$filter=ListPrice gt 100000
$top	number	Limits the size of the result set. Default is 10, maximum is 200.
Limit results from the Test dataset to only 2
https://api.bridgedataoutput.com/api/v2/OData/dataset_id/Properties?access_token=access_token&$top=2
$orderby	string	Response field to sort query by (either “desc” or “asc”)
Sort order by descending price
https://api.bridgedataoutput.com/api/v2/OData/dataset_id/Properties?access_token=access_token&$orderby=ListPrice desc
$expand	string	Include query specified entities inline with response
Expand the relevant listing agent for a given property
https://api.bridgedataoutput.com/api/v2/OData/dataset_id/Properties?access_token=access_token&$expand=ListAgent
Query Functions
Function	Description
any	Search fields where any element of an array is satisfied by a condition
Return listings where there is an option of electric heating
https://api.bridgedataoutput.com/api/v2/OData/dataset_id/Properties?access_token=access_token&$filter=Heating/any(a: a eq 'Electric')
all	Search fields where all elements of an array is satisfied by a condition
Return listings where all of the flooring is hardwood
https://api.bridgedataoutput.com/api/v2/OData/dataset_id/Properties?access_token=access_token&$filter=Flooring/all(a: a eq 'Hardwood')
geo.distance	Search by coordinates
Return listings that are near specific co-ordinates, to a radius of 0.5 miles
https://api.bridgedataoutput.com/api/v2/OData/dataset_id/Properties?access_token=access_token&$filter=geo.distance(Coordinates, POINT(-118.62 34.22)) lt 0.5
geo.intersects	If you know the extents of a polygonal region, you can provide the each point as co-ordinates
Return listings within a shape
https://api.bridgedataoutput.com/api/v2/OData/dataset_id/Properties?access_token=access_token&$filter=geo.intersects(Coordinates, POLYGON((-127.02 45.08,-127.02 45.38,-127.32 45.38,-127.32 45.08,-127.02 45.08)))
tolower	Search fields with lowercase queries
Return listings using a lowercase query
https://api.bridgedataoutput.com/api/v2/OData/dataset_id/Properties?access_token=access_token&$filter=tolower(StandardStatus) eq 'active'
startswith	Search fields by a string prefix
Return listings using a city that starts with a specific string
https://api.bridgedataoutput.com/api/v2/OData/dataset_id/Properties?access_token=access_token&$filter=startswith(City, 'Spring')
endswith	Search fields by string ending
Return listings using a city that ends with a specific string
https://api.bridgedataoutput.com/api/v2/OData/dataset_id/Properties?access_token=access_token&$filter=endswith(City, 'field')
contains	Search field by string inclusion
Return listings using a city that contains a specific string
https://api.bridgedataoutput.com/api/v2/OData/dataset_id/Properties?access_token=access_token&$filter=contains(City, 'nge')
date	Search fields by date
Return listings with a specific date
https://api.bridgedataoutput.com/api/v2/OData/dataset_id/Properties?access_token=access_token&$filter=date(ModificationTimestamp) eq 2017-08-29
time	Search fields by time
Return listings with a specific time
https://api.bridgedataoutput.com/api/v2/OData/dataset_id/Properties?access_token=access_token&$filter=time(ModificationTimestamp) eq 17:03:04
year	Search fields by year
Return listings with a specific year
https://api.bridgedataoutput.com/api/v2/OData/dataset_id/Properties?access_token=access_token&$filter=year(ModificationTimestamp) eq 2017
month	Search fields by month
Return listings with a specific month
https://api.bridgedataoutput.com/api/v2/OData/dataset_id/Properties?access_token=access_token&$filter=month(ModificationTimestamp) eq 12
day	Search fields by day
Return listings with a specific day
https://api.bridgedataoutput.com/api/v2/OData/dataset_id/Properties?access_token=access_token&$filter=day(ModificationTimestamp) eq 23
hour	Search fields by hour
Return listings with a specific hour
https://api.bridgedataoutput.com/api/v2/OData/dataset_id/Properties?access_token=access_token&$filter=hour(ModificationTimestamp) eq 17
now()	Search fields by current timestamp
Return listings within the current timestamp
https://api.bridgedataoutput.com/api/v2/OData/dataset_id/Properties?access_token=access_token&$filter=ModificationTimestamp eq now()
Examples
Search for a specific property by ListingId
https://api.bridgedataoutput.com/api/v2/OData/dataset_id/Property?access_token=access_token&$filter=ListingId eq ‘123456789’
Search for a property by address
https://api.bridgedataoutput.com/api/v2/OData/dataset_id/Property?access_token=access_token&$filter=UnparsedAddress eq ‘123 Main’
Search for a property by address, using ‘tolower’ to work around API case-sensitivity
https://api.bridgedataoutput.com/api/v2/OData/dataset_id/Property?access_token=access_token&$filter=tolower(UnparsedAddress) eq ‘123 main’
Search for all residential properties that are on sale
https://api.bridgedataoutput.com/api/v2/OData/dataset_id/Property?access_token=access_token&$filter=PropertyType eq ‘Residential’ and StandardStatus eq ‘Active’
Search for all residential properties that are for rent
https://api.bridgedataoutput.com/api/v2/OData/dataset_id/Property?access_token=access_token&$filter=PropertyType eq ‘Residential Income’ and StandardStatus eq ‘Active’
Search for all residential properties that are for sale and in a specific zip code
https://api.bridgedataoutput.com/api/v2/OData/dataset_id/Property?access_token=access_token&$filter=PropertyType eq ‘Residential Lease’ and PostalCode eq ‘90210’ and StandardStatus eq ‘Active’
Search for all properties that are in one of two zip codes
https://api.bridgedataoutput.com/api/v2/OData/dataset_id/Property?access_token=access_token&$filter=PostalCode eq ‘12345’ or PostalCode eq ‘54321’
Search using complex nested queries
https://api.bridgedataoutput.com/api/v2/OData/dataset_id/Property?access_token=access_token&$filter=((InternetEntireListingDisplayYN ne false) and ((StandardStatus eq ‘Closed’) and (((YearBuilt eq null) or ((YearBuilt le 1986) and (YearBuilt ge 1976))) and (((LivingArea eq null) or ((LivingArea le 3264) and (LivingArea ge 2412))) and ((CloseDate ge 2019-09-01) and (((BedroomsTotal eq null) or ((BedroomsTotal le 5) and (BedroomsTotal ge 3))) and (((BathroomsTotalDecimal eq null) or ((BathroomsTotalDecimal le 3.5) and (BathroomsTotalDecimal ge 1.5))) and ((SpecialListingConditions ne ‘Auction’) and ((SpecialListingConditions ne ‘Probate’) and ((SpecialListingConditions ne ‘Short Sale’) and ((SpecialListingConditions ne ‘REO’) and (((ClosePrice eq null) or ((ClosePrice le 472666) and (ClosePrice ge 315110))) and (geo.distance(Coordinates,POINT(-115.10998 36.091513)) lt 0.5)))))))))))))
