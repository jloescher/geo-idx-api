<?php

namespace App\Http\Controllers\Marketing;

use App\Http\Controllers\Controller;
use Illuminate\Contracts\View\View;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Request;

class SalesPageController extends Controller
{
    public function __invoke(Request $request): View|RedirectResponse
    {
        return view('marketing.home');
    }
}
