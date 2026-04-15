export type TabName = 'feed' | 'stories';

type Props = {
  active: TabName;
  onChange: (tab: TabName) => void;
};

export default function Tabs({ active, onChange }: Props) {
  return (
    <nav className="tabs" role="tablist">
      <button
        type="button"
        role="tab"
        aria-selected={active === 'feed'}
        className={`tab${active === 'feed' ? ' active' : ''}`}
        onClick={() => onChange('feed')}
      >
        Feed
      </button>
      <button
        type="button"
        role="tab"
        aria-selected={active === 'stories'}
        className={`tab${active === 'stories' ? ' active' : ''}`}
        onClick={() => onChange('stories')}
      >
        Stories
      </button>
    </nav>
  );
}
