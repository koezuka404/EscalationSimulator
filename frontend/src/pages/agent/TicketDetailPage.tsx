import { useParams } from "react-router-dom";
import { PageFrame } from "../../components/common/PageFrame";
import { TicketView } from "../../components/ticket/TicketView";
import { paths } from "../../paths";

export function TicketDetailPage() {
  const { ticketId } = useParams();
  return (
    <PageFrame title="対応中のチケット">
      {ticketId && <TicketView id={ticketId} backTo={paths.queue} backLabel="待ち順へ戻る" />}
    </PageFrame>
  );
}
