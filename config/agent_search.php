<?php

return [

    /*
    |--------------------------------------------------------------------------
    | Drawn geometry safeguards (map search)
    |--------------------------------------------------------------------------
    |
    | Limits abusive mega-shapes while allowing metro-scale polygons. Span is
    | max(lat span, lng span) of the ring axis-aligned bounding box in degrees.
    |
    */
    'max_polygon_bbox_span_deg' => (float) env('AGENT_SEARCH_MAX_POLYGON_BBOX_SPAN_DEG', 0.85),

    'max_circle_radius_m' => (float) env('AGENT_SEARCH_MAX_CIRCLE_RADIUS_M', 75000),

];
