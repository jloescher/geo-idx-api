<?php

namespace App\Http\Controllers\AgentPortal;

use App\Http\Controllers\Controller;
use App\Http\Requests\AgentPortal\AgentAlertTemplateUpsertRequest;
use App\Models\AgentAlertTemplate;
use App\Models\User;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\DB;

class AgentAlertTemplateController extends Controller
{
    public function index(Request $request): JsonResponse
    {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        $rows = AgentAlertTemplate::query()
            ->where('user_id', $user->id)
            ->latest('id')
            ->get()
            ->map(fn (AgentAlertTemplate $template): array => [
                'id' => $template->id,
                'name' => $template->name,
                'template_type' => $template->template_type,
                'body_json' => $template->body_json,
                'schedule_json' => $template->schedule_json,
                'usage_count' => $template->usage_count ?? 0,
                'last_used_at' => $template->last_used_at?->toIso8601String(),
                'created_at' => $template->created_at?->toIso8601String(),
                'updated_at' => $template->updated_at?->toIso8601String(),
            ]);

        return response()->json(['data' => $rows]);
    }

    public function store(AgentAlertTemplateUpsertRequest $request): JsonResponse
    {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        $validated = $request->validated();
        $audit = $this->appendAuditEntry([], 'created', $user->id);

        $row = AgentAlertTemplate::query()->create([
            'user_id' => $user->id,
            'name' => (string) $validated['name'],
            'template_type' => (string) $validated['template_type'],
            'body_json' => $validated['body_json'] ?? [],
            'schedule_json' => $validated['schedule_json'] ?? null,
            'usage_count' => 0,
            'audit_json' => $audit,
        ]);

        return response()->json(['data' => $row], 201);
    }

    public function update(AgentAlertTemplateUpsertRequest $request, int $templateId): JsonResponse
    {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        $row = AgentAlertTemplate::query()
            ->where('id', $templateId)
            ->where('user_id', $user->id)
            ->firstOrFail();

        $validated = $request->validated();
        $existingAudit = is_array($row->audit_json) ? $row->audit_json : [];
        $audit = $this->appendAuditEntry($existingAudit, 'updated', $user->id);

        $row->update([
            'name' => (string) $validated['name'],
            'template_type' => (string) $validated['template_type'],
            'body_json' => $validated['body_json'] ?? [],
            'schedule_json' => $validated['schedule_json'] ?? $row->schedule_json,
            'audit_json' => $audit,
        ]);

        return response()->json(['data' => $row]);
    }

    public function destroy(Request $request, int $templateId): JsonResponse
    {
        /** @var User|null $user */
        $user = $request->user();
        abort_if($user === null, 401);

        $template = AgentAlertTemplate::query()
            ->where('id', $templateId)
            ->where('user_id', $user->id)
            ->firstOrFail();

        DB::table('agent_alert_templates')->where('id', $template->id)->delete();

        return response()->json([], 204);
    }

    /**
     * @param  list<array<string, mixed>>  $entries
     * @return list<array<string, mixed>>
     */
    private function appendAuditEntry(array $entries, string $action, int $userId): array
    {
        $entries[] = [
            'action' => $action,
            'user_id' => $userId,
            'timestamp' => now()->toIso8601String(),
        ];

        return $entries;
    }
}
