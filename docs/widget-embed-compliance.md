# Quantyra GeoIDX Widget Embed Guide

## Single Script Embed

```html
<script
  src="https://geo.quantyralabs.com/js/widgets/loader.js?token=SUBSCRIBER_TOKEN&primaryColor=3b82f6&accentColor=10b981"
  data-widget="search"
  data-footer-required="true"
  async
></script>
<div data-quantyragidx-footer="true"></div>
```

## GHL Installation Notes

- Complete OAuth install and URL registration before generating widget key.
- Use only approved domains from the registered URL list.
- Keep a footer anchor on every page where any non-footer widget is mounted.
- Use `window.QuantyraGeoIDX.initWidget(type, config)` for programmatic widget initialization.

## Compliance Checklist

- Rule 13: Brokerage name is rendered in footer disclosures.
- Rule 22: Listing brokerage + listing attribution remain required adjacent to listing details.
- Rule 23: MLS source attribution is rendered in the footer disclosure block.
- Rule 24: Consumer disclaimer text is rendered in the footer disclosure block with timestamp.
- Rule 29: Anti-scraping notice is included in footer disclosures.
- Rule 30: DMCA notice + contact email is displayed in footer disclosures.
- Rule 28: Seller contact fields must never be rendered by widget templates.
- Rule 25: Map popups must link to detailed views that carry full required disclosures.
