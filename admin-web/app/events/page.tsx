import RefreshButton from "@/components/RefreshButton";
import { fetchJSON } from "@/lib/api";
import type { EventRow } from "@/lib/types";

export default async function EventsPage() {
  const events = await fetchJSON<EventRow[]>("/v1/events/recent");

  return (
    <div>
      <div className="mb-4 flex items-center justify-between">
        <h1 className="text-xl font-semibold">Recent Events</h1>
        <RefreshButton />
      </div>
      <div className="overflow-x-auto rounded border border-slate-200 bg-white">
        <table className="w-full text-left text-sm">
          <thead className="border-b border-slate-200 bg-slate-100 text-slate-600">
            <tr>
              <th className="px-3 py-2">Event</th>
              <th className="px-3 py-2">User</th>
              <th className="px-3 py-2">Platform</th>
              <th className="px-3 py-2">Campaign</th>
              <th className="px-3 py-2">Click ID</th>
              <th className="px-3 py-2">Received</th>
            </tr>
          </thead>
          <tbody>
            {events.map((event) => (
              <tr key={event.id} className="border-b border-slate-100 last:border-0">
                <td className="px-3 py-2 font-medium">{event.event_name}</td>
                <td className="px-3 py-2">{event.user_id}</td>
                <td className="px-3 py-2">{event.ad_platform}</td>
                <td className="px-3 py-2">{event.campaign_id}</td>
                <td className="px-3 py-2 text-slate-500">{event.click_id}</td>
                <td className="px-3 py-2 text-slate-500">
                  {new Date(event.received_at).toLocaleString()}
                </td>
              </tr>
            ))}
            {events.length === 0 && (
              <tr>
                <td colSpan={6} className="px-3 py-6 text-center text-slate-400">
                  No events yet.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
