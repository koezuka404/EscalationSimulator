import { useParams } from "react-router-dom";
import { PageFrame } from "../../components/common/PageFrame";
import { TicketView } from "../../components/ticket/TicketView";
import { paths } from "../../paths";

export function TicketDetailPage() {
  const { ticketId } = useParams();
  return (
    <PageFrame title="チケットの詳細">
      {ticketId && <TicketView id={ticketId} backTo={paths.adminTickets} backLabel="全件へ戻る" />}
    </PageFrame>
  );
}
