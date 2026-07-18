import type { GenerationJob } from "@/entities/generation-job/model";
import { GenerationFailureRecoveryCard } from "./GenerationFailureRecoveryCard";
import { GenerationProgressCard } from "./GenerationProgressCard";

type GenerationJobStatusCardProps = {
  job: GenerationJob;
  canCancel?: boolean;
  isCancelling?: boolean;
  isRetrying?: boolean;
  onCancel?: () => void;
  onRetry?: () => void;
  onSimplerRequest?: () => void;
  onReloadLatest?: () => void;
  onKeepCurrent?: () => void;
  onEditDetails?: () => void;
};

export function GenerationJobStatusCard({
  job,
  canCancel = false,
  isCancelling = false,
  isRetrying = false,
  onCancel,
  onRetry,
  onSimplerRequest,
  onReloadLatest,
  onKeepCurrent,
  onEditDetails
}: GenerationJobStatusCardProps) {
  if (job.status === "failed") {
    return (
      <GenerationFailureRecoveryCard
        isRetrying={isRetrying}
        job={job}
        onEditDetails={onEditDetails}
        onKeepCurrent={onKeepCurrent}
        onReloadLatest={onReloadLatest}
        onRetry={onRetry}
        onSimplerRequest={onSimplerRequest}
      />
    );
  }
  return <GenerationProgressCard canCancel={canCancel} isCancelling={isCancelling} job={job} onCancel={onCancel} />;
}
