import { Router, NavigationEnd, ActivatedRouteSnapshot } from '@angular/router';
import { Location } from '@angular/common';
import { Observable } from "rxjs";
import 'rxjs/add/operator/distinctUntilChanged'
import 'rxjs/add/operator/filter'
import 'rxjs/add/operator/map'
import 'rxjs/add/operator/startWith'
import 'rxjs/add/operator/shareReplay'

export function getListRoute(routes: ActivatedRouteSnapshot[]): ActivatedRouteSnapshot {
    for (let route of routes) {
        if ("primary" in route.data) {
            return route;
        }

        let r = getListRoute(route.children);
        if (r != null) {
            return r;
        }
    }

    return null;
}

export function articleRoute(router: Router): Observable<ActivatedRouteSnapshot> {
    return router.events.filter(event =>
        event instanceof NavigationEnd
    ).startWith(null).map(v => {
        return getArticleRoute([router.routerState.snapshot.root])
    }).distinctUntilChanged().shareReplay(1);
}

export function getArticleRoute(routes: ActivatedRouteSnapshot[]): ActivatedRouteSnapshot {
    for (let route of routes) {
        if ("articleID" in route.params) {
            return route;
        }

        let r = getArticleRoute(route.children);
        if (r != null) {
            return r;
        }
    }

    return null;
}