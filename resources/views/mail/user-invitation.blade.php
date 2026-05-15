<x-mail::message>
# {{ __('You are invited to GeoIDX') }}

{{ __('Use the button below to create your subscriber account. This link expires on :date.', ['date' => $expiresAt->timezone(config('app.timezone'))->toDayDateTimeString()]) }}

<x-mail::button :url="$acceptUrl">
{{ __('Accept invitation') }}
</x-mail::button>

{{ __('If you did not expect this message, you can ignore it.') }}

{{ __('Thanks,') }}<br>
{{ config('app.name') }}
</x-mail::message>
