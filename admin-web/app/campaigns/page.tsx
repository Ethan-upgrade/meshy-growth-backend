import { revalidatePath } from "next/cache";
import { fetchJSON } from "@/lib/api";
import type { Campaign } from "@/lib/types";

const ADMIN_API_URL = process.env.ADMIN_API_URL ?? "http://admin-api:8080";

async function createCampaign(formData: FormData) {
  "use server";

  const payload = {
    name: String(formData.get("name") ?? ""),
    ad_platform: String(formData.get("ad_platform") ?? ""),
    external_campaign_id: String(formData.get("external_campaign_id") ?? ""),
    attribution_window_days: Number(formData.get("attribution_window_days") ?? 7),
  };

  const res = await fetch(`${ADMIN_API_URL}/v1/campaigns`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  if (!res.ok) {
    throw new Error(`failed to create campaign: ${res.status}`);
  }

  revalidatePath("/campaigns");
}

export default async function CampaignsPage() {
  const campaigns = await fetchJSON<Campaign[]>("/v1/campaigns");

  return (
    <div>
      <h1 className="mb-4 text-xl font-semibold">Campaigns</h1>

      <div className="mb-8 overflow-x-auto rounded border border-slate-200 bg-white">
        <table className="w-full text-left text-sm">
          <thead className="border-b border-slate-200 bg-slate-100 text-slate-600">
            <tr>
              <th className="px-3 py-2">Name</th>
              <th className="px-3 py-2">Platform</th>
              <th className="px-3 py-2">External ID</th>
              <th className="px-3 py-2">Attribution Window</th>
              <th className="px-3 py-2">Active</th>
            </tr>
          </thead>
          <tbody>
            {campaigns.map((campaign) => (
              <tr key={campaign.id} className="border-b border-slate-100 last:border-0">
                <td className="px-3 py-2 font-medium">{campaign.name}</td>
                <td className="px-3 py-2">{campaign.ad_platform}</td>
                <td className="px-3 py-2 text-slate-500">{campaign.external_campaign_id}</td>
                <td className="px-3 py-2">{campaign.attribution_window_days}d</td>
                <td className="px-3 py-2">{campaign.is_active ? "Yes" : "No"}</td>
              </tr>
            ))}
            {campaigns.length === 0 && (
              <tr>
                <td colSpan={5} className="px-3 py-6 text-center text-slate-400">
                  No campaigns yet.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      <div className="rounded border border-slate-200 bg-white p-4">
        <h2 className="mb-3 text-sm font-semibold text-slate-600">New Campaign</h2>
        <form action={createCampaign} className="grid grid-cols-2 gap-4 sm:grid-cols-4">
          <input
            name="name"
            placeholder="Name"
            required
            className="rounded border border-slate-300 px-3 py-2 text-sm"
          />
          <select
            name="ad_platform"
            required
            className="rounded border border-slate-300 px-3 py-2 text-sm"
          >
            <option value="meta">meta</option>
            <option value="google">google</option>
            <option value="tiktok">tiktok</option>
          </select>
          <input
            name="external_campaign_id"
            placeholder="External Campaign ID"
            required
            className="rounded border border-slate-300 px-3 py-2 text-sm"
          />
          <input
            name="attribution_window_days"
            type="number"
            defaultValue={7}
            min={1}
            required
            className="rounded border border-slate-300 px-3 py-2 text-sm"
          />
          <button
            type="submit"
            className="col-span-2 rounded bg-slate-900 px-3 py-2 text-sm text-white hover:bg-slate-700 sm:col-span-1"
          >
            Create
          </button>
        </form>
      </div>
    </div>
  );
}
