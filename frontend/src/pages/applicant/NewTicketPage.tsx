import { useNavigate } from "react-router-dom";
import { useAuth } from "../../auth/AuthProvider";
import { NewTicketForm } from "../../components/applicant/NewTicketForm";
import { PageFrame } from "../../components/common/PageFrame";
import { paths } from "../../paths";

export function NewTicketPage() {
  const { user } = useAuth();
  const navigate = useNavigate();
  return (
    <PageFrame title="起票">
      <NewTicketForm locked={!user?.customer_id} onCreated={(ticket) => navigate(paths.ticket(ticket.id))} />
    </PageFrame>
  );
}
