import { DemoForm } from "../../components/admin/DemoForm";
import { PageFrame } from "../../components/common/PageFrame";
import { useLive } from "../../websocket/connection";

export function DemoPage() {
  const live = useLive();
  return (
    <PageFrame title="デモ">
      <p className="mb-4 max-w-xl text-sm text-stone-600">
        顧客に結びついた申請者の名前で、通常の起票と同じように問い合わせを作ります。実行中は1件だけです。
      </p>
      <DemoForm progress={live.demo} onStarted={(run) => live.setDemo(run)} onStopped={(run) => live.setDemo(run)} />
    </PageFrame>
  );
}
