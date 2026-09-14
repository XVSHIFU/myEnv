import python from './assets/icons/python.svg';
import node from './assets/icons/nodejs.svg';
import java from './assets/icons/java.svg';
import go from './assets/icons/go.svg';
import rust from './assets/icons/rust.svg';

const brands: Record<string, string> = {python, node, java, go, rust};

// Original brand colors stay intact. Adjacent visible language names supply the accessible label.
export function BrandIcon({tool}: {tool: string}) {
    const src = Object.hasOwn(brands, tool) ? brands[tool] : undefined;
    if (!src) return null;
    return <span className={'language-mark brand-icon brand-icon-' + tool} aria-hidden="true"><img src={src} width={24} height={24} alt="" draggable={false}/></span>;
}
