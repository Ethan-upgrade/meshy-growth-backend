import RefreshButton from "@/components/RefreshButton";
import { fetchJSON } from "@/lib/api";
import type { ConversionRow, QueueStatus } from "@/lib/types";

type StatusSummary = Record<string, number>;

function statusColor(status: string) {
  if (status === "success") return "text-green-600";
  if (status === "failed") return "text-red-600";
  return "text-slate-500";
}

export default async function ConversionsPage() {
  const [summary, recent, queue] = await Promise.all([
    fetchJSON<StatusSummary>("/v1/conversions/summary"),
    fetchJSON<ConversionRow[]>("/v1/conversions/recent"),
    fetchJSON<QueueStatus>("/v1/queue/status"),
  ]);

  return (
    <div>
      <div className="mb-4 flex items-center justify-between">
        <h1 className="text-xl font-semibold">Conversion Callbacks</h1>
        <RefreshButton />
      </div>

      <div className="mb-6 grid grid-cols-2 gap-4 sm:grid-cols-4">
        {Object.entries(summary).map(([status, count]) => (
          <div key={status} className="rounded border border-slate-200 bg-white p-4">
            <div className={`text-2xl font-semibold ${statusColor(status)}`}>{count}</div>
            <div className="text-sm text-slate-500">{status}</div>
          </div>
        ))}
      </div>

      <div className="mb-6">
        <h2 className="mb-2 text-sm font-semibold text-slate-600">
          Retry Queue Status (asynq — full detail at localhost:8090)
        </h2>
        <div className="grid grid-cols-3 gap-4 sm:grid-cols-6">
          {Object.entries(queue).map(([key, value]) => (
            <div key={key} className="rounded border border-slate-200 bg-white p-3 text-center">
              <div className="text-lg font-semibold">{value}</div>
              <div className="text-xs text-slate-500">{key}</div>
            </div>
          ))}
        </div>
      </div>

      <div className="overflow-x-auto rounded border border-slate-200 bg-white">
        <table className="w-full text-left text-sm">
          <thead className="border-b border-slate-200 bg-slate-100 text-slate-600">
            <tr>
              <th className="px-3 py-2">Platform</th>
              <th className="px-3 py-2">Campaign</th>
              <th className="px-3 py-2">Click ID</th>
              <th className="px-3 py-2">Status</th>
              <th className="px-3 py-2">Attempts</th>
              <th className="px-3 py-2">Last Error</th>
              <th className="px-3 py-2">Updated</th>
            </tr>
          </thead>
          <tbody>
            {recent.map((conversion) => (
              <tr key={conversion.id} className="border-b border-slate-100 last:border-0">
                <td className="px-3 py-2">{conversion.ad_platform}</td>
                <td className="px-3 py-2">{conversion.campaign_id}</td>
                <td className="px-3 py-2 text-slate-500">{conversion.click_id}</td>
                <td className={`px-3 py-2 font-medium ${statusColor(conversion.status)}`}>
                  {conversion.status}
                </td>
                <td className="px-3 py-2">{conversion.attempt_count}</td>
                <td className="px-3 py-2 text-slate-500">{conversion.last_error}</td>
                <td className="px-3 py-2 text-slate-500">
                  {new Date(conversion.updated_at).toLocaleString()}
                </td>
              </tr>
            ))}
            {recent.length === 0 && (
              <tr>
                <td colSpan={7} className="px-3 py-6 text-center text-slate-400">
                  No conversions yet.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
