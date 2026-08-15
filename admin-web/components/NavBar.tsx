import Link from "next/link";

const links = [
  { href: "/events", label: "Events" },
  { href: "/conversions", label: "Conversions" },
  { href: "/experiments", label: "Experiments" },
  { href: "/campaigns", label: "Campaigns" },
];

export default function NavBar() {
  return (
    <header className="border-b border-slate-200">
      <div className="mx-auto flex max-w-6xl items-center gap-6 px-6 py-4">
        <span className="font-semibold">Growth Admin</span>
        <nav className="flex gap-4 text-sm text-slate-600">
          {links.map((link) => (
            <Link key={link.href} href={link.href} className="hover:text-slate-900">
              {link.label}
            </Link>
          ))}
        </nav>
      </div>
    </header>
  );
}
