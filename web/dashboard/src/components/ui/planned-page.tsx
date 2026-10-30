import { PageHeader } from "./page-header";
import { Planned } from "./states";

/** A route that is scoped but not yet implemented — honest, never a dead link. */
export function PlannedPage({ title, subtitle, phase, detail }: {
  title: string;
  subtitle?: string;
  phase: string;
  detail?: string;
}) {
  return (
    <div>
      <PageHeader title={title} subtitle={subtitle} />
      <Planned phase={phase} detail={detail} />
    </div>
  );
}
