import Link from "next/link";
import { fetchJSON } from "@/lib/api";
import type { ExperimentDetail, ExperimentStatsResponse } from "@/lib/types";

export default async function ExperimentDetailPage({
  params,
}: {
  params: Promise<{ key: string }>;
}) {
  const { key } = await params;
  const [experiment, stats] = await Promise.all([
    fetchJSON<ExperimentDetail>(`/v1/experiments/${key}`),
    fetchJSON<ExperimentStatsResponse>(`/v1/experiments/${key}/stats?goal_event=purchase`),
  ]);

  const statsByVariant = new Map(stats.variants.map((row) => [row.variant_key, row]));

  return (
    <div>
      <Link href="/experiments" className="text-sm text-blue-600 hover:underline">
        &larr; Back to experiments
      </Link>
      <h1 className="mb-1 mt-2 text-xl font-semibold">{experiment.name}</h1>
      <p className="mb-4 text-sm text-slate-500">
        {experiment.key} &middot; {experiment.status} &middot; goal event: purchase
      </p>

      {stats.srm_check?.flagged && (
        <div className="mb-4 rounded border border-red-300 bg-red-50 px-3 py-2 text-sm text-red-800">
          Sample Ratio Mismatch detected: observed assignment split diverges from configured
          weights (chi-square {stats.srm_check.chi_square.toFixed(2)}, p={" "}
          {stats.srm_check.p_value.toFixed(4)}). Assignment/bucketing logic may have a bug —
          treat results below with caution.
        </div>
      )}

      <div className="overflow-x-auto rounded border border-slate-200 bg-white">
        <table className="w-full text-left text-sm">
          <thead className="border-b border-slate-200 bg-slate-100 text-slate-600">
            <tr>
              <th className="px-3 py-2">Variant</th>
              <th className="px-3 py-2">Weight</th>
              <th className="px-3 py-2">Assigned</th>
              <th className="px-3 py-2">Converted</th>
              <th className="px-3 py-2">Conversion Rate</th>
              <th className="px-3 py-2">95% CI</th>
              <th className="px-3 py-2">p-value vs control</th>
            </tr>
          </thead>
          <tbody>
            {experiment.variants.map((variant, idx) => {
              const row = statsByVariant.get(variant.key);
              const assigned = row?.assigned ?? 0;
              const converted = row?.converted ?? 0;
              const rate = row ? (row.conversion_rate * 100).toFixed(1) : "0.0";
              const ci = row?.confidence_interval;
              return (
                <tr key={variant.key} className="border-b border-slate-100 last:border-0">
                  <td className="px-3 py-2 font-medium">
                    {variant.name}
                    {idx === 0 && (
                      <span className="ml-1 text-xs font-normal text-slate-400">(control)</span>
                    )}
                  </td>
                  <td className="px-3 py-2">{variant.weight}</td>
                  <td className="px-3 py-2">{assigned}</td>
                  <td className="px-3 py-2">{converted}</td>
                  <td className="px-3 py-2">{rate}%</td>
                  <td className="px-3 py-2">
                    {ci ? `${(ci[0] * 100).toFixed(1)}% – ${(ci[1] * 100).toFixed(1)}%` : "—"}
                  </td>
                  <td className="px-3 py-2">
                    {row?.p_value_vs_control !== undefined
                      ? row.p_value_vs_control.toFixed(4)
                      : "—"}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
}
