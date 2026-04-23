<?php

namespace App\Http\Controllers\Marketing;

use App\Http\Controllers\Controller;
use Illuminate\Contracts\View\View;

class SalesPageController extends Controller
{
    public function __invoke(): View
    {
        return view('marketing.sales');
    }
}
