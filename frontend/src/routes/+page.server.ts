import { redirect } from '@sveltejs/kit';

/** 根路径直接跳到工作台，避免客户端 onMount+goto 与水合/路由初始化竞态（hash 报错） */
export function load() {
	throw redirect(307, '/workbench');
}
