import Link from "next/link";
import { fetchJSON } from "@/lib/api";
import type { ExperimentSummary } from "@/lib/types";

export default async function ExperimentsPage() {
  const experiments = await fetchJSON<ExperimentSummary[]>("/v1/experiments");

  return (
    <div>
      <h1 className="mb-4 text-xl font-semibold">Experiments</h1>
      <div className="overflow-x-auto rounded border border-slate-200 bg-white">
        <table className="w-full text-left text-sm">
          <thead className="border-b border-slate-200 bg-slate-100 text-slate-600">
            <tr>
              <th className="px-3 py-2">Key</th>
              <th className="px-3 py-2">Name</th>
              <th className="px-3 py-2">Status</th>
              <th className="px-3 py-2">Traffic Allocation</th>
            </tr>
          </thead>
          <tbody>
            {experiments.map((experiment) => (
              <tr key={experiment.id} className="border-b border-slate-100 last:border-0">
                <td className="px-3 py-2 font-medium">
                  <Link href={`/experiments/${experiment.key}`} className="text-blue-600 hover:underline">
                    {experiment.key}
                  </Link>
                </td>
                <td className="px-3 py-2">{experiment.name}</td>
                <td className="px-3 py-2">{experiment.status}</td>
                <td className="px-3 py-2">{experiment.traffic_allocation}</td>
              </tr>
            ))}
            {experiments.length === 0 && (
              <tr>
                <td colSpan={4} className="px-3 py-6 text-center text-slate-400">
                  No experiments yet.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
