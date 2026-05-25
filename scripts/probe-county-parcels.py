#!/usr/bin/env python3
"""Probe Florida county parcel ArcGIS endpoints for MLS coverage counties."""

import json
import ssl
import urllib.parse
import urllib.request

PRIMARY = [
    ("alachua", "stellar", "https://gis.floridahealth.gov/server/rest/services/EHWATER/Parcels/MapServer/0/query", (-82.40, 29.60, -82.30, 29.70), None),
    ("charlotte", "stellar", "https://agis3.charlottecountyfl.gov/arcgis/rest/services/Essentials/CCGISLayers/MapServer/27/query", (-82.10, 26.90, -82.00, 27.00), None),
    ("desoto", "stellar", "https://www45.swfwmd.state.fl.us/arcgis12/rest/services/BaseVector/parcel_search/MapServer/3/query", (-81.90, 27.20, -81.80, 27.30), None),
    ("flagler", "stellar", "https://gis.palmcoast.gov/hosting/rest/services/External/FlaglerCountyParcels/MapServer/1/query", (-81.30, 29.45, -81.20, 29.55), None),
    ("hillsborough", "stellar", "https://maps.hillsboroughcounty.org/arcgis/rest/services/InfoLayers/HC_ParcelsPublic/FeatureServer/0/query", (-82.50, 27.90, -82.40, 28.00), None),
    ("lake", "stellar", "https://gis.lakecountyfl.gov/lakegis/rest/services/OpenData/OpenData1/FeatureServer/12/query", (-81.75, 28.75, -81.65, 28.85), None),
    ("manatee", "stellar", "https://www.mymanatee.org/gisits/rest/services/commonoperational/parcellines/MapServer/0/query", (-82.60, 27.45, -82.50, 27.55), None),
    ("marion", "stellar", "https://gis.marionfl.org/public/rest/services/General/Parcels/MapServer/0/query", (-82.15, 29.15, -82.05, 29.25), None),
    ("okeechobee", "stellar", "https://geoweb.sfwmd.gov/agsext2/rest/services/LandOwnershipAndInterests/NormalizedParcels/FeatureServer/0/query", (-80.95, 27.35, -80.85, 27.45), "CNTYNAME='Okeechobee'"),
    ("orange", "stellar", "https://services2.arcgis.com/N4cKzJ9dzXmsPNRs/ArcGIS/rest/services/orange_county_parcels/FeatureServer/0/query", (-81.38, 28.44, -81.36, 28.46), None),
    ("osceola", "stellar", "https://services2.arcgis.com/V2PQwgZMTFfgM0Xu/arcgis/rest/services/Osceola_County_FL_WFL1/FeatureServer/0/query", (-81.20, 28.05, -81.10, 28.15), None),
    ("pasco", "stellar", "https://maps.pascopa.com/arcgis/rest/services/Parcels/MapServer/3/query", (-82.55, 28.30, -82.45, 28.40), None),
    ("pinellas", "stellar", "https://www45.swfwmd.state.fl.us/arcgis12/rest/services/BaseVector/parcel_search/MapServer/13/query", (-82.85, 27.95, -82.75, 28.05), None),
    ("polk", "stellar", "https://gis.polk-county.net/hosting/rest/services/TPO/TPO_Parcel_and_Permit_Map/MapServer/1/query", (-81.75, 28.00, -81.65, 28.10), None),
    ("sarasota", "stellar", "https://services3.arcgis.com/icrWMv7eBkctFu1f/arcgis/rest/services/ParcelHosted/FeatureServer/0/query", (-82.50, 27.30, -82.40, 27.40), None),
    ("sumter", "stellar", "https://gis.ecfrpc.org/arcgis/rest/services/Basemap/MapServer/4/query", (-82.10, 28.65, -82.00, 28.75), None),
    ("volusia", "stellar", "https://maps5.vcgov.org/arcgis/rest/services/Open_Data/Open_Data_3/FeatureServer/36/query", (-81.20, 29.05, -81.10, 29.15), None),
    ("broward", "beaches", "https://services5.arcgis.com/wI5GZmCtnUU8ueya/ArcGIS/rest/services/Broward_County_Parcel_Boundary/FeatureServer/1/query", (-80.20, 26.10, -80.10, 26.20), None),
    ("palm-beach", "beaches", "https://maps.co.palm-beach.fl.us/arcgis/rest/services/OpenData/open_data_v2/MapServer/0/query", (-80.10, 26.70, -80.00, 26.80), None),
    ("martin", "beaches", "https://geoweb.sfwmd.gov/agsext2/rest/services/LandOwnershipAndInterests/NormalizedParcels/FeatureServer/0/query", (-80.45, 27.05, -80.35, 27.15), "CNTYNAME='Martin'"),
    ("st-lucie", "beaches", "https://geoweb.sfwmd.gov/agsext2/rest/services/LandOwnershipAndInterests/NormalizedParcels/FeatureServer/0/query", (-80.50, 27.35, -80.40, 27.45), "CNTYNAME='St Lucie'"),
    ("miami-dade", "beaches", "https://gisweb.miamidade.gov/arcgis/rest/services/MD_LandInformation/MapServer/26/query", (-80.25, 25.75, -80.15, 25.85), None),
]

FALLBACK = [
    ("pinellas", "stellar", "https://egis.pinellas.gov/pcpagis/rest/services/PcpaBaseMap/BaseMapParcelAerials/MapServer/167/query", (-82.85, 27.95, -82.75, 28.05), None),
    ("pinellas", "stellar", "https://egis.pinellascounty.org/arcgis/rest/services/PARCEL/MapServer/0/query", (-82.85, 27.95, -82.75, 28.05), None),
    ("orange", "stellar", "https://services2.arcgis.com/N4cKzJ9dzXmsPNRs/ArcGIS/rest/services/orange_county_parcels/FeatureServer/0/query", (-81.40, 28.50, -81.30, 28.60), None),
]

ctx = ssl.create_default_context()


def probe(name, mls, url, bbox, where, timeout=35):
    w, s, e, n = bbox
    params = {
        "f": "json",
        "geometry": f"{w},{s},{e},{n}",
        "geometryType": "esriGeometryEnvelope",
        "inSR": "4326",
        "spatialRel": "esriSpatialRelIntersects",
        "returnCountOnly": "true",
    }
    if where:
        params["where"] = where
    full = f"{url}?{urllib.parse.urlencode(params)}"
    try:
        req = urllib.request.Request(full, headers={"User-Agent": "idx-api-probe/1.0"})
        with urllib.request.urlopen(req, timeout=timeout, context=ctx) as resp:
            body = resp.read().decode()
    except Exception as ex:
        return {"county": name, "mls": mls, "status": "FAIL", "detail": str(ex)[:80], "url": url}
    try:
        doc = json.loads(body)
    except json.JSONDecodeError:
        return {"county": name, "mls": mls, "status": "BAD", "detail": "non-json", "url": url}
    if "error" in doc:
        err = doc["error"]
        details = err.get("details") or []
        msg = details[0] if details else err.get("message", "error")
        return {"county": name, "mls": mls, "status": "ERR", "detail": msg[:80], "url": url}
    cnt = doc.get("count")
    if cnt is None:
        return {"county": name, "mls": mls, "status": "UNK", "detail": "no count", "url": url}
    status = "OK" if cnt > 0 else "ZERO"
    return {"county": name, "mls": mls, "status": status, "detail": f"count={cnt}", "url": url}


print("=== PRIMARY sources (verified probe) ===")
for row in PRIMARY:
    r = probe(*row)
    print(f"{r['county']:14} {r['mls']:8} {r['status']:5}  {r['detail']}")

print("\n=== FALLBACK / notes ===")
for row in FALLBACK:
    r = probe(*row, timeout=45 if "pinellascounty" in row[2] else 35)
    print(f"{r['county']:14} {r['mls']:8} {r['status']:5}  {r['detail']}  ({row[2].split('/')[2]})")

print("\n=== FDOR statewide (do not use) ===")
for label, where in [("bbox-only", None), ("CO_NO=52", "CO_NO=52")]:
    r = probe("fdor", "n/a", "https://services9.arcgis.com/Gh9awoU677aKree0/ArcGIS/rest/services/Florida_Statewide_Cadastral/FeatureServer/0/query", (-82.85, 27.95, -82.84, 27.96), where, timeout=90)
    print(f"fdor-{label:10} {r['status']:5}  {r['detail']}")
