# Structured Data Patterns

When to use: Adding JSON-LD schema to comparison pages, alternative suggestion pages, or plan detail pages for rich search results.

## Patterns

### Software Application Schema for Plans

```php
// In Blade template
<script type="application/ld+json">
{
  "@context": "https://schema.org",
  "@type": "SoftwareApplication",
  "name": "GeoIDX {{ $plan['name'] }}",
  "offers": {
    "@type": "Offer",
    "price": "{{ $plan[$interval] }}",
    "priceCurrency": "USD",
    "priceValidUntil": "{{ now()->addYear()->format('Y-m-d') }}"
  },
  "aggregateRating": {
    "@type": "AggregateRating",
    "ratingValue": "4.8",
    "ratingCount": "{{ $reviewCount }}"
  }
}
</script>
```

### Comparison Table Schema

```php
<script type="application/ld+json">
{
  "@context": "https://schema.org",
  "@type": "Table",
  "about": "GeoIDX Plan Comparison",
  "mainEntity": {
    "@type": "ItemList",
    "itemListElement": [
      @foreach($plans as $plan)
      {
        "@type": "ListItem",
        "position": {{ $loop->iteration }},
        "name": "{{ $plan['name'] }}",
        "item": {
          "@type": "Product",
          "name": "GeoIDX {{ $plan['name'] }}"
        }
      }@if(!$loop->last),@endif
      @endforeach
    ]
  }
}
</script>
```

### FAQ Schema for Alternatives Pages

Use when addressing "What is the difference between X and Y?" queries:

```php
"mainEntity": [
  @foreach($faqs as $faq)
  {
    "@type": "Question",
    "name": "{{ $faq['question'] }}",
    "acceptedAnswer": {
      "@type": "Answer",
      "text": "{{ $faq['answer'] }}"
    }
  }@if(!$loop->last),@endif
  @endforeach
]
```

## Pitfalls

Do not include dynamic pricing in schema without updating `priceValidUntil`—stale structured data triggers Google Shopping policy violations. Use `AggregateOffer` only when displaying multiple billing intervals simultaneously.

-----